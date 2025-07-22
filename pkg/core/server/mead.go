package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/AudiusProject/audiusd/pkg/api/core/v1beta1"
	"github.com/AudiusProject/audiusd/pkg/api/ddex/v1beta2"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/core/db"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	"google.golang.org/protobuf/proto"
)

var (
	// MEAD top level errors
	ErrMEADMessageValidation   = errors.New("MEAD message validation failed")
	ErrMEADMessageFinalization = errors.New("MEAD message finalization failed")

	// Create MEAD message validation errors
	ErrMEADAddressNotEmpty   = errors.New("MEAD address is not empty")
	ErrMEADFromAddressEmpty  = errors.New("MEAD from address is empty")
	ErrMEADToAddressNotEmpty = errors.New("MEAD to address is not empty")
	ErrMEADNonceNotOne       = errors.New("MEAD nonce is not one")
)

func (s *Server) finalizeMEAD(ctx context.Context, req *abcitypes.FinalizeBlockRequest, txhash string, tx *v1beta1.Transaction, messageIndex int64) error {
	if len(tx.Envelope.Messages) <= int(messageIndex) {
		return fmt.Errorf("message index out of range")
	}

	mead := tx.Envelope.Messages[messageIndex].GetMead()
	if mead == nil {
		return fmt.Errorf("tx: %s, message index: %d, MEAD message not found", txhash, messageIndex)
	}

	switch mead.Header.ControlType {
	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_NEW_MESSAGE:
		if err := s.validateMEADNewMessage(ctx, mead); err != nil {
			return errors.Join(ErrMEADMessageValidation, err)
		}
		if err := s.finalizeMEADNewMessage(ctx, req, txhash, messageIndex, mead); err != nil {
			return errors.Join(ErrMEADMessageFinalization, err)
		}
		return nil

	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_UPDATED_MESSAGE:
		if err := s.validateMEADUpdateMessage(ctx, mead); err != nil {
			return errors.Join(ErrMEADMessageValidation, err)
		}
		if err := s.finalizeMEADUpdateMessage(ctx, req, txhash, messageIndex, mead); err != nil {
			return errors.Join(ErrMEADMessageFinalization, err)
		}
		return nil

	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_TAKEDOWN_MESSAGE:
		if err := s.validateMEADTakedownMessage(ctx, mead); err != nil {
			return errors.Join(ErrMEADMessageValidation, err)
		}
		if err := s.finalizeMEADTakedownMessage(ctx, req, txhash, messageIndex, mead); err != nil {
			return errors.Join(ErrMEADMessageFinalization, err)
		}
		return nil

	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_UNSPECIFIED:
		return fmt.Errorf("tx: %s, message index: %d, MEAD message control type is unspecified", txhash, messageIndex)

	default:
		return fmt.Errorf("tx: %s, message index: %d, unsupported MEAD message control type: %s", txhash, messageIndex, mead.Header.ControlType)
	}
}

/** MEAD New Message */

// Validate a MEAD message that's expected to be a NEW_MESSAGE, expects that the transaction header is valid
func (s *Server) validateMEADNewMessage(_ context.Context, mead *v1beta2.MediaEnrichmentDescription) error {
	if mead.Address != "" {
		return ErrMEADAddressNotEmpty
	}

	if mead.Header.From == "" {
		return ErrMEADFromAddressEmpty
	}

	if mead.Header.To != "" {
		return ErrMEADToAddressNotEmpty
	}

	if mead.Header.Nonce != 1 {
		return ErrMEADNonceNotOne
	}

	// TODO: add validation for conflicts and duplicates
	return nil
}

func (s *Server) finalizeMEADNewMessage(ctx context.Context, req *abcitypes.FinalizeBlockRequest, txhash string, messageIndex int64, mead *v1beta2.MediaEnrichmentDescription) error {
	nonce := fmt.Sprintf("%d", mead.Header.Nonce)
	// the MEAD address is the location of the message on the chain
	meadAddress := common.CreateAddress(mead, s.config.GenesisFile.ChainID, req.Height, nonce)

	rawMessage, err := proto.Marshal(mead)
	if err != nil {
		return fmt.Errorf("failed to marshal MEAD message: %w", err)
	}

	// Create acknowledgment for potential use in responses
	ack := &v1beta2.MediaEnrichmentDescriptionAck{
		MeadAddress: meadAddress,
		Nonce:       mead.Header.Nonce,
	}

	rawAcknowledgment, err := proto.Marshal(ack)
	if err != nil {
		return fmt.Errorf("failed to marshal MEAD acknowledgment: %w", err)
	}

	qtx := s.getDb()
	if err := qtx.InsertCoreMEAD(ctx, db.InsertCoreMEADParams{
		TxHash:             txhash,
		Index:              messageIndex,
		Address:            meadAddress,
		Sender:             mead.Header.From,
		Nonce:              int64(mead.Header.Nonce),
		MessageControlType: int16(mead.Header.ControlType),
		ResourceAddresses:  mead.ResourceAddresses,
		ReleaseAddresses:   mead.ReleaseAddresses,
		RawMessage:         rawMessage,
		RawAcknowledgment:  rawAcknowledgment,
		BlockHeight:        req.Height,
	}); err != nil {
		return fmt.Errorf("failed to insert MEAD: %w", err)
	}

	return nil
}

/** MEAD Update Message */

func (s *Server) validateMEADUpdateMessage(_ context.Context, mead *v1beta2.MediaEnrichmentDescription) error {
	return nil
}

func (s *Server) finalizeMEADUpdateMessage(ctx context.Context, req *abcitypes.FinalizeBlockRequest, txhash string, messageIndex int64, mead *v1beta2.MediaEnrichmentDescription) error {
	return nil
}

/** MEAD Takedown Message */

func (s *Server) validateMEADTakedownMessage(_ context.Context, mead *v1beta2.MediaEnrichmentDescription) error {
	return nil
}

func (s *Server) finalizeMEADTakedownMessage(ctx context.Context, req *abcitypes.FinalizeBlockRequest, txhash string, messageIndex int64, mead *v1beta2.MediaEnrichmentDescription) error {
	return nil
}
