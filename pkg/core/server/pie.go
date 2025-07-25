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
	// PIE top level errors
	ErrPIEMessageValidation   = errors.New("PIE message validation failed")
	ErrPIEMessageFinalization = errors.New("PIE message finalization failed")

	// Create PIE message validation errors
	ErrPIEAddressNotEmpty   = errors.New("PIE address is not empty")
	ErrPIEFromAddressEmpty  = errors.New("PIE from address is empty")
	ErrPIEToAddressNotEmpty = errors.New("PIE to address is not empty")
	ErrPIENonceNotOne       = errors.New("PIE nonce is not one")

	// Update PIE message validation errors
	ErrPIEAddressEmpty   = errors.New("PIE address is empty")
	ErrPIEToAddressEmpty = errors.New("PIE to address is empty")
	ErrPIEAddressNotTo   = errors.New("PIE address is not the target of the message")
	ErrPIENonceNotNext   = errors.New("PIE nonce is not the next nonce")
)

func (s *Server) finalizePIE(ctx context.Context, req *abcitypes.FinalizeBlockRequest, txhash string, tx *v1beta1.Transaction, messageIndex int64) error {
	if len(tx.Envelope.Messages) <= int(messageIndex) {
		return fmt.Errorf("message index out of range")
	}

	pie := tx.Envelope.Messages[messageIndex].GetPie()
	if pie == nil {
		return fmt.Errorf("tx: %s, message index: %d, PIE message not found", txhash, messageIndex)
	}

	switch pie.Header.ControlType {
	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_NEW_MESSAGE:
		if err := s.validatePIENewMessage(ctx, pie); err != nil {
			return errors.Join(ErrPIEMessageValidation, err)
		}
		if err := s.finalizePIENewMessage(ctx, req, txhash, messageIndex, pie); err != nil {
			return errors.Join(ErrPIEMessageFinalization, err)
		}
		return nil

	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_UPDATED_MESSAGE:
		if err := s.validatePIEUpdateMessage(ctx, pie); err != nil {
			return errors.Join(ErrPIEMessageValidation, err)
		}
		if err := s.finalizePIEUpdateMessage(ctx, req, txhash, messageIndex, pie); err != nil {
			return errors.Join(ErrPIEMessageFinalization, err)
		}
		return nil

	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_TAKEDOWN_MESSAGE:
		if err := s.validatePIETakedownMessage(ctx, pie); err != nil {
			return errors.Join(ErrPIEMessageValidation, err)
		}
		if err := s.finalizePIETakedownMessage(ctx, req, txhash, messageIndex, pie); err != nil {
			return errors.Join(ErrPIEMessageFinalization, err)
		}
		return nil

	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_UNSPECIFIED:
		return fmt.Errorf("tx: %s, message index: %d, PIE message control type is unspecified", txhash, messageIndex)

	default:
		return fmt.Errorf("tx: %s, message index: %d, unsupported PIE message control type: %s", txhash, messageIndex, pie.Header.ControlType)
	}
}

/** PIE New Message */

// Validate a PIE message that's expected to be a NEW_MESSAGE, expects that the transaction header is valid
func (s *Server) validatePIENewMessage(_ context.Context, pie *v1beta2.PartyIdentificationEnrichment) error {
	if pie.Address != "" {
		return ErrPIEAddressNotEmpty
	}

	if pie.Header.From == "" {
		return ErrPIEFromAddressEmpty
	}

	if pie.Header.To != "" {
		return ErrPIEToAddressNotEmpty
	}

	if pie.Header.Nonce != 1 {
		return ErrPIENonceNotOne
	}

	// TODO: add validation for conflicts and duplicates
	return nil
}

func (s *Server) finalizePIENewMessage(ctx context.Context, req *abcitypes.FinalizeBlockRequest, txhash string, messageIndex int64, pie *v1beta2.PartyIdentificationEnrichment) error {
	nonce := fmt.Sprintf("%d", pie.Header.Nonce)
	// the PIE address is the location of the message on the chain
	pieAddress := common.CreateAddress(pie, s.config.GenesisFile.ChainID, req.Height, nonce)

	rawMessage, err := proto.Marshal(pie)
	if err != nil {
		return fmt.Errorf("failed to marshal PIE message: %w", err)
	}

	// Create acknowledgment for potential use in responses
	ack := &v1beta2.PartyIdentificationEnrichmentAck{
		PieAddress: pieAddress,
		Nonce:      pie.Header.Nonce,
	}

	rawAcknowledgment, err := proto.Marshal(ack)
	if err != nil {
		return fmt.Errorf("failed to marshal PIE acknowledgment: %w", err)
	}

	qtx := s.getDb()
	if err := qtx.InsertCorePIE(ctx, db.InsertCorePIEParams{
		TxHash:             txhash,
		Index:              messageIndex,
		Address:            pieAddress,
		Sender:             pie.Header.From,
		Nonce:              int64(pie.Header.Nonce),
		MessageControlType: int16(pie.Header.ControlType),
		PartyAddresses:     pie.PartyAddresses,
		RawMessage:         rawMessage,
		RawAcknowledgment:  rawAcknowledgment,
		BlockHeight:        req.Height,
	}); err != nil {
		return fmt.Errorf("failed to insert PIE: %w", err)
	}

	return nil
}

/** PIE Update Message */

func (s *Server) validatePIEUpdateMessage(ctx context.Context, pie *v1beta2.PartyIdentificationEnrichment) error {
	if pie.Address == "" {
		return ErrPIEAddressEmpty
	}

	if pie.Header.From == "" {
		return ErrPIEFromAddressEmpty
	}

	if pie.Header.To == "" {
		return ErrPIEToAddressEmpty
	}

	// address of the PIE must also be the target of the message
	if pie.Address != pie.Header.To {
		return ErrPIEAddressNotTo
	}

	storedPIE, err := s.db.GetPIE(ctx, pie.Address)
	if err != nil {
		return fmt.Errorf("failed to get stored PIE: %w", err)
	}

	if storedPIE.Nonce != int64(pie.Header.Nonce-1) {
		return ErrPIENonceNotNext
	}

	// TODO: validate party addresses and their contents

	return nil
}

func (s *Server) finalizePIEUpdateMessage(ctx context.Context, req *abcitypes.FinalizeBlockRequest, txhash string, messageIndex int64, pie *v1beta2.PartyIdentificationEnrichment) error {
	if err := s.validatePIEUpdateMessage(ctx, pie); err != nil {
		return errors.Join(ErrPIEMessageValidation, err)
	}

	rawMessage, err := proto.Marshal(pie)
	if err != nil {
		return fmt.Errorf("failed to marshal PIE message: %w", err)
	}

	ack := &v1beta2.PartyIdentificationEnrichmentAck{
		PieAddress: pie.Address,
		Nonce:      pie.Header.Nonce,
	}

	rawAcknowledgment, err := proto.Marshal(ack)
	if err != nil {
		return fmt.Errorf("failed to marshal PIE acknowledgment: %w", err)
	}

	qtx := s.getDb()
	// same insert as a new PIE, the nonce differentiates the update from the original PIE
	if err := qtx.InsertCorePIE(ctx, db.InsertCorePIEParams{
		TxHash:             txhash,
		Index:              messageIndex,
		Address:            pie.Address,
		Sender:             pie.Header.From,
		Nonce:              int64(pie.Header.Nonce),
		MessageControlType: int16(pie.Header.ControlType),
		PartyAddresses:     pie.PartyAddresses,
		RawMessage:         rawMessage,
		RawAcknowledgment:  rawAcknowledgment,
		BlockHeight:        req.Height,
	}); err != nil {
		return fmt.Errorf("failed to insert PIE: %w", err)
	}

	return nil
}

/** PIE Takedown Message */

func (s *Server) validatePIETakedownMessage(_ context.Context, _ *v1beta2.PartyIdentificationEnrichment) error {
	return nil
}

func (s *Server) finalizePIETakedownMessage(_ context.Context, _ *abcitypes.FinalizeBlockRequest, _ string, _ int64, _ *v1beta2.PartyIdentificationEnrichment) error {
	return nil
}
