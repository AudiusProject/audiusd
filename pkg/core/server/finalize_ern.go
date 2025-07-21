package server

import (
	"context"
	"fmt"

	"github.com/AudiusProject/audiusd/pkg/api/core/v1beta1"
	"github.com/AudiusProject/audiusd/pkg/api/ddex/v1beta2"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/core/db"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	"google.golang.org/protobuf/proto"
)

func (s *Server) finalizeV2Transaction(ctx context.Context, req *abcitypes.FinalizeBlockRequest, tx *v1beta1.Transaction) error {
	header := tx.Envelope.Header
	if header.ChainId != s.config.GenesisFile.ChainID {
		return fmt.Errorf("invalid chain id: %s", header.ChainId)
	}

	if header.Expiration < req.Height {
		return fmt.Errorf("transaction expired")
	}

	var err error
	for _, msg := range tx.Envelope.Messages {
		switch msg.Message.(type) {
		case *v1beta1.Message_Ern:
			ern := msg.GetErn()
			err = s.finalizeERN(ctx, req, tx, ern)
			// TODO: Add support for MEAD and PIE when they are enabled in the core API
			// case *v1beta1.Message_Mead:
			//     mead := msg.GetMead()
			//     err = s.finalizeMEAD(ctx, req, tx, mead)
			// case *v1beta1.Message_Pie:
			//     pie := msg.GetPie()
			//     err = s.finalizePIE(ctx, req, tx, pie)
		}

		if err != nil {
			return fmt.Errorf("failed to finalize message: %w", err)
		}
	}

	return nil
}

func (s *Server) finalizeERN(ctx context.Context, req *abcitypes.FinalizeBlockRequest, tx *v1beta1.Transaction, ern *v1beta2.ElectronicReleaseNotification) error {
	// Use envelope nonce as string for address generation
	envelopeNonce := tx.Envelope.Header.Nonce

	ernAddress := common.CreateAddress(ern, s.config.GenesisFile.ChainID, req.Height, envelopeNonce)

	// Collect all addresses
	partyAddresses := make([]string, len(ern.PartyList))
	for i, party := range ern.PartyList {
		partyAddresses[i] = common.CreateAddress(party, s.config.GenesisFile.ChainID, req.Height, envelopeNonce)
	}

	resourceAddresses := make([]string, len(ern.ResourceList))
	for i, resource := range ern.ResourceList {
		resourceAddresses[i] = common.CreateAddress(resource, s.config.GenesisFile.ChainID, req.Height, envelopeNonce)
	}

	releaseAddresses := make([]string, len(ern.ReleaseList))
	for i, release := range ern.ReleaseList {
		releaseAddresses[i] = common.CreateAddress(release, s.config.GenesisFile.ChainID, req.Height, envelopeNonce)
	}

	dealAddresses := make([]string, len(ern.DealList))
	for i, deal := range ern.DealList {
		dealAddresses[i] = common.CreateAddress(deal, s.config.GenesisFile.ChainID, req.Height, envelopeNonce)
	}

	// Marshal the entire ERN message
	rawMessage, err := proto.Marshal(ern)
	if err != nil {
		return fmt.Errorf("failed to marshal ERN message: %w", err)
	}

	// TODO: Recover sender address from transaction signature
	senderAddress := ern.Header.From

	// Handle different message control types - use ERN header nonce for database storage
	switch ern.Header.ControlType {
	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_NEW_MESSAGE:
		err = s.finalizeERNCreate(ctx, req, ernAddress, senderAddress, ern.Header.Nonce, int16(ern.Header.ControlType), partyAddresses, resourceAddresses, releaseAddresses, dealAddresses, rawMessage, req.Height)
	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_UPDATED_MESSAGE:
		err = s.finalizeERNUpdate(ctx, req, ernAddress, senderAddress, ern.Header.Nonce, int16(ern.Header.ControlType), partyAddresses, resourceAddresses, releaseAddresses, dealAddresses, rawMessage, req.Height)
	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_TAKEDOWN_MESSAGE:
		err = s.finalizeERNTakedown(ctx, req, ernAddress, senderAddress, ern.Header.Nonce, int16(ern.Header.ControlType), partyAddresses, resourceAddresses, releaseAddresses, dealAddresses, rawMessage, req.Height)
	default:
		return fmt.Errorf("unsupported ERN message control type: %v", ern.Header.ControlType)
	}

	return err
}

func (s *Server) finalizeERNCreate(ctx context.Context, req *abcitypes.FinalizeBlockRequest, address, sender string, nonce uint64, messageControlType int16, partyAddresses, resourceAddresses, releaseAddresses, dealAddresses []string, rawMessage []byte, blockHeight int64) error {
	qtx := s.getDb()

	// Convert nonce to int64 for database storage
	dbNonce := int64(nonce)

	return qtx.InsertCoreERN(ctx, db.InsertCoreERNParams{
		Address:            address,
		Sender:             sender,
		Nonce:              dbNonce,
		MessageControlType: messageControlType,
		PartyAddresses:     partyAddresses,
		ResourceAddresses:  resourceAddresses,
		ReleaseAddresses:   releaseAddresses,
		DealAddresses:      dealAddresses,
		RawMessage:         rawMessage,
		BlockHeight:        blockHeight,
	})
}

func (s *Server) finalizeERNUpdate(ctx context.Context, req *abcitypes.FinalizeBlockRequest, address, sender string, nonce uint64, messageControlType int16, partyAddresses, resourceAddresses, releaseAddresses, dealAddresses []string, rawMessage []byte, blockHeight int64) error {
	// TODO: Implement ERN update logic
	// For now, just insert the updated ERN (you may want to update existing records instead)
	return s.finalizeERNCreate(ctx, req, address, sender, nonce, messageControlType, partyAddresses, resourceAddresses, releaseAddresses, dealAddresses, rawMessage, blockHeight)
}

func (s *Server) finalizeERNTakedown(ctx context.Context, req *abcitypes.FinalizeBlockRequest, address, sender string, nonce uint64, messageControlType int16, partyAddresses, resourceAddresses, releaseAddresses, dealAddresses []string, rawMessage []byte, blockHeight int64) error {
	// TODO: Implement ERN takedown logic
	// For now, just insert the takedown ERN (you may want to mark existing records as taken down)
	return s.finalizeERNCreate(ctx, req, address, sender, nonce, messageControlType, partyAddresses, resourceAddresses, releaseAddresses, dealAddresses, rawMessage, blockHeight)
}

// TODO: Implement when MEAD support is added to core API
// func (s *Server) finalizeMEAD(ctx context.Context, req *abcitypes.FinalizeBlockRequest, tx *v1beta1.Transaction, mead *v1beta2.MediaEnrichmentDescription) error {
//     // Similar implementation for MEAD messages
//     return nil
// }

// TODO: Implement when PIE support is added to core API
// func (s *Server) finalizePIE(ctx context.Context, req *abcitypes.FinalizeBlockRequest, tx *v1beta1.Transaction, pie *v1beta2.PartyIdentificationEnrichment) error {
//     // Similar implementation for PIE messages
//     return nil
// }
