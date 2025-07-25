package server

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/AudiusProject/audiusd/pkg/api/core/v1beta1"
	"github.com/AudiusProject/audiusd/pkg/api/ddex/v1beta2"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/core/db"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	"google.golang.org/protobuf/proto"
)

var (
	// ERN top level errors
	ErrERNMessageValidation   = errors.New("ERN message validation failed")
	ErrERNMessageFinalization = errors.New("ERN message finalization failed")

	// Create ERN message validation errors
	ErrERNAddressNotEmpty   = errors.New("ERN address is not empty")
	ErrERNFromAddressEmpty  = errors.New("ERN from address is empty")
	ErrERNToAddressNotEmpty = errors.New("ERN to address is not empty")
	ErrERNNonceNotOne       = errors.New("ERN nonce is not one")

	// Update ERN message validation errors
	ErrERNAddressEmpty   = errors.New("ERN address is empty")
	ErrERNToAddressEmpty = errors.New("ERN to address is empty")
	ErrERNAddressNotTo   = errors.New("ERN address is not the target of the message")
	ErrERNNonceNotNext   = errors.New("ERN nonce is not the next nonce")
)

func (s *Server) finalizeERN(ctx context.Context, req *abcitypes.FinalizeBlockRequest, txhash string, tx *v1beta1.Transaction, messageIndex int64) error {
	if len(tx.Envelope.Messages) <= int(messageIndex) {
		return fmt.Errorf("message index out of range")
	}

	ern := tx.Envelope.Messages[messageIndex].GetErn()
	if ern == nil {
		return fmt.Errorf("tx: %s, message index: %d, ERN message not found", txhash, messageIndex)
	}

	switch ern.Header.ControlType {
	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_NEW_MESSAGE:
		if err := s.validateERNNewMessage(ctx, ern); err != nil {
			return errors.Join(ErrERNMessageValidation, err)
		}
		if err := s.finalizeERNNewMessage(ctx, req, txhash, messageIndex, ern); err != nil {
			return errors.Join(ErrERNMessageFinalization, err)
		}
		return nil

	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_UPDATED_MESSAGE:
		if err := s.validateERNUpdateMessage(ctx, ern); err != nil {
			return errors.Join(ErrERNMessageValidation, err)
		}
		if err := s.finalizeERNUpdateMessage(ctx, req, txhash, messageIndex, ern); err != nil {
			return errors.Join(ErrERNMessageFinalization, err)
		}
		return nil
	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_TAKEDOWN_MESSAGE:
		if err := s.validateERNTakedownMessage(ctx, ern); err != nil {
			return errors.Join(ErrERNMessageValidation, err)
		}
		if err := s.finalizeERNTakedownMessage(ctx, req, txhash, messageIndex, ern); err != nil {
			return errors.Join(ErrERNMessageFinalization, err)
		}
		return nil
	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_UNSPECIFIED:
		return fmt.Errorf("tx: %s, message index: %d, ERN message control type is unspecified", txhash, messageIndex)
	default:
		return fmt.Errorf("tx: %s, message index: %d, unsupported ERN message control type: %s", txhash, messageIndex, ern.Header.ControlType)
	}
}

/** ERN New Message */

// Validate an ERN message that's expected to be a NEW_MESSAGE, expects that the transaction header is valid
func (s *Server) validateERNNewMessage(_ context.Context, ern *v1beta2.ElectronicReleaseNotification) error {
	if ern.Address != "" {
		return ErrERNAddressNotEmpty
	}

	if ern.Header.From == "" {
		return ErrERNFromAddressEmpty
	}

	if ern.Header.To != "" {
		return ErrERNToAddressNotEmpty
	}

	if ern.Header.Nonce != 1 {
		return ErrERNNonceNotOne
	}

	// TODO: add validation for conflicts and duplicates
	return nil
}

func (s *Server) finalizeERNNewMessage(ctx context.Context, req *abcitypes.FinalizeBlockRequest, txhash string, messageIndex int64, ern *v1beta2.ElectronicReleaseNotification) error {
	nonce := fmt.Sprintf("%d", ern.Header.Nonce)
	// the ERN address is the location of the message on the chain
	ernAddress := common.CreateAddress(ern, s.config.GenesisFile.ChainID, req.Height, nonce)

	// Collect all addresses, all underlying objects use the same source ERN nonce
	partyAddresses := make([]string, len(ern.PartyList))
	for i, party := range ern.PartyList {
		partyAddresses[i] = common.CreateAddress(party, s.config.GenesisFile.ChainID, req.Height, nonce)
	}

	resourceAddresses := make([]string, len(ern.ResourceList))
	for i, resource := range ern.ResourceList {
		resourceAddresses[i] = common.CreateAddress(resource, s.config.GenesisFile.ChainID, req.Height, nonce)
	}

	releaseAddresses := make([]string, len(ern.ReleaseList))
	for i, release := range ern.ReleaseList {
		releaseAddresses[i] = common.CreateAddress(release, s.config.GenesisFile.ChainID, req.Height, nonce)
	}

	dealAddresses := make([]string, len(ern.DealList))
	for i, deal := range ern.DealList {
		dealAddresses[i] = common.CreateAddress(deal, s.config.GenesisFile.ChainID, req.Height, nonce)
	}

	rawMessage, err := proto.Marshal(ern)
	if err != nil {
		return fmt.Errorf("failed to marshal ERN message: %w", err)
	}

	ack := &v1beta2.ElectronicReleaseNotificationAck{
		ErnAddress:        ernAddress,
		Nonce:             ern.Header.Nonce,
		PartyAddresses:    partyAddresses,
		ResourceAddresses: resourceAddresses,
		ReleaseAddresses:  releaseAddresses,
		DealAddresses:     dealAddresses,
	}

	rawAcknowledgment, err := proto.Marshal(ack)
	if err != nil {
		return fmt.Errorf("failed to marshal ERN acknowledgment: %w", err)
	}

	qtx := s.getDb()
	if err := qtx.InsertCoreERN(ctx, db.InsertCoreERNParams{
		TxHash:             txhash,
		Index:              messageIndex,
		Address:            ernAddress,
		Sender:             ern.Header.From,
		Nonce:              int64(ern.Header.Nonce),
		MessageControlType: int16(ern.Header.ControlType),
		PartyAddresses:     partyAddresses,
		ResourceAddresses:  resourceAddresses,
		ReleaseAddresses:   releaseAddresses,
		DealAddresses:      dealAddresses,
		RawMessage:         rawMessage,
		RawAcknowledgment:  rawAcknowledgment,
		BlockHeight:        req.Height,
	}); err != nil {
		return fmt.Errorf("failed to insert ERN: %w", err)
	}

	return nil
}

/** ERN Update Message */

// TODO: profile this function
func (s *Server) validateERNUpdateMessage(ctx context.Context, ern *v1beta2.ElectronicReleaseNotification) error {
	if ern.Address == "" {
		return ErrERNAddressEmpty
	}

	if ern.Header.From == "" {
		return ErrERNFromAddressEmpty
	}

	if ern.Header.To == "" {
		return ErrERNToAddressEmpty
	}

	// address of the ERN must also be the target of the message
	if ern.Address != ern.Header.To {
		return ErrERNAddressNotTo
	}

	storedERN, err := s.db.GetERN(ctx, ern.Address)
	if err != nil {
		return fmt.Errorf("failed to get stored ERN: %w", err)
	}

	if storedERN.Nonce != int64(ern.Header.Nonce-1) {
		return ErrERNNonceNotNext
	}

	// TODO: validate party, resource, release, deal addresses and their contents
	// ensure that entities with provided addresses exist in this ERN
	for _, party := range ern.PartyList {
		if party.Address != "" {
			if !slices.Contains(storedERN.PartyAddresses, party.Address) {
				return fmt.Errorf("party address %s not found in ERN", party.Address)
			}
		}
	}

	for _, resource := range ern.ResourceList {

		if resource.Address != "" {
			if !slices.Contains(storedERN.ResourceAddresses, resource.Address) {
				return fmt.Errorf("resource address %s not found in ERN", resource.Address)
			}
		}
	}

	for _, release := range ern.ReleaseList {
		if release.Address != "" {
			if !slices.Contains(storedERN.ReleaseAddresses, release.Address) {
				return fmt.Errorf("release address %s not found in ERN", release.Address)
			}
		}
	}

	for _, deal := range ern.DealList {
		if deal.Address != "" {
			if !slices.Contains(storedERN.DealAddresses, deal.Address) {
				return fmt.Errorf("deal address %s not found in ERN", deal.Address)
			}
		}
	}

	return nil
}

func (s *Server) finalizeERNUpdateMessage(ctx context.Context, req *abcitypes.FinalizeBlockRequest, txhash string, messageIndex int64, ern *v1beta2.ElectronicReleaseNotification) error {
	if err := s.validateERNUpdateMessage(ctx, ern); err != nil {
		return errors.Join(ErrERNMessageValidation, err)
	}

	nonce := fmt.Sprintf("%d", ern.Header.Nonce)

	// create new addresses for new entities, otherwise they will be the same as the original addresses
	partyAddresses := make([]string, len(ern.PartyList))
	for i, party := range ern.PartyList {
		if party.Address == "" {
			partyAddresses[i] = common.CreateAddress(party, s.config.GenesisFile.ChainID, req.Height, nonce)
		} else {
			partyAddresses[i] = party.Address
		}
	}

	resourceAddresses := make([]string, len(ern.ResourceList))
	for i, resource := range ern.ResourceList {
		if resource.Address == "" {
			resourceAddresses[i] = common.CreateAddress(resource, s.config.GenesisFile.ChainID, req.Height, nonce)
		} else {
			resourceAddresses[i] = resource.Address
		}
	}

	releaseAddresses := make([]string, len(ern.ReleaseList))
	for i, release := range ern.ReleaseList {
		if release.Address == "" {
			releaseAddresses[i] = common.CreateAddress(release, s.config.GenesisFile.ChainID, req.Height, nonce)
		} else {
			releaseAddresses[i] = release.Address
		}
	}

	dealAddresses := make([]string, len(ern.DealList))
	for i, deal := range ern.DealList {
		if deal.Address == "" {
			dealAddresses[i] = common.CreateAddress(deal, s.config.GenesisFile.ChainID, req.Height, nonce)
		} else {
			dealAddresses[i] = deal.Address
		}
	}

	rawMessage, err := proto.Marshal(ern)
	if err != nil {
		return fmt.Errorf("failed to marshal ERN message: %w", err)
	}

	ack := &v1beta2.ElectronicReleaseNotificationAck{
		ErnAddress:        ern.Address,
		Nonce:             ern.Header.Nonce,
		PartyAddresses:    partyAddresses,
		ResourceAddresses: resourceAddresses,
		ReleaseAddresses:  releaseAddresses,
		DealAddresses:     dealAddresses,
	}

	rawAcknowledgment, err := proto.Marshal(ack)
	if err != nil {
		return fmt.Errorf("failed to marshal ERN acknowledgment: %w", err)
	}

	qtx := s.getDb()
	// same insert as a new ERN, the nonce differentiates the update from the original ERN
	if err := qtx.InsertCoreERN(ctx, db.InsertCoreERNParams{
		TxHash:             txhash,
		Index:              messageIndex,
		Address:            ern.Address,
		Sender:             ern.Header.From,
		Nonce:              int64(ern.Header.Nonce),
		MessageControlType: int16(ern.Header.ControlType),
		PartyAddresses:     partyAddresses,
		ResourceAddresses:  resourceAddresses,
		ReleaseAddresses:   releaseAddresses,
		DealAddresses:      dealAddresses,
		RawMessage:         rawMessage,
		RawAcknowledgment:  rawAcknowledgment,
		BlockHeight:        req.Height,
	}); err != nil {
		return fmt.Errorf("failed to insert ERN: %w", err)
	}

	return nil
}

/** ERN Takedown Message */

func (s *Server) validateERNTakedownMessage(_ context.Context, _ *v1beta2.ElectronicReleaseNotification) error {
	return nil
}

func (s *Server) finalizeERNTakedownMessage(ctx context.Context, _ *abcitypes.FinalizeBlockRequest, _ string, _ int64, ern *v1beta2.ElectronicReleaseNotification) error {
	if err := s.validateERNTakedownMessage(ctx, ern); err != nil {
		return errors.Join(ErrERNMessageValidation, err)
	}

	_, err := s.getDb().GetERNCreate(ctx, ern.Address)
	if err != nil {
		return fmt.Errorf("failed to get original ERN: %w", err)
	}
	return nil
}
