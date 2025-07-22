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

	// Calculate transaction hash for receipt
	txhash := s.toTxHash(tx.Envelope)

	s.logger.Debugf("finalizing v2 transaction %s with %d messages", txhash, len(tx.Envelope.Messages))

	var err error
	for i, msg := range tx.Envelope.Messages {
		switch msg.Message.(type) {
		case *v1beta1.Message_Ern:
			ern := msg.GetErn()
			err = s.finalizeERN(ctx, req, txhash, tx, int64(i), ern)
			if err != nil {
				return fmt.Errorf("failed to finalize ERN message: %w", err)
			}
		case *v1beta1.Message_Mead:
			mead := msg.GetMead()
			err = s.finalizeMEAD(ctx, req, txhash, int64(i), tx, mead)
			if err != nil {
				return fmt.Errorf("failed to finalize MEAD message: %w", err)
			}
		case *v1beta1.Message_Pie:
			pie := msg.GetPie()
			err = s.finalizePIE(ctx, req, txhash, int64(i), tx, pie)
			if err != nil {
				return fmt.Errorf("failed to finalize PIE message: %w", err)
			}
		}

		if err != nil {
			return fmt.Errorf("failed to finalize message: %w", err)
		}
	}
	return nil
}

func (s *Server) finalizeERN(ctx context.Context, req *abcitypes.FinalizeBlockRequest, txhash string, tx *v1beta1.Transaction, messageIndex int64, ern *v1beta2.ElectronicReleaseNotification) error {
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

	// TODO: Recover sender address from transaction signature
	senderAddress := ern.Header.From

	// Handle different message control types - use ERN header nonce for database storage
	switch ern.Header.ControlType {
	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_NEW_MESSAGE:
		err = s.finalizeERNCreate(ctx, req, txhash, messageIndex, ernAddress, senderAddress, ern.Header.Nonce, int16(ern.Header.ControlType), partyAddresses, resourceAddresses, releaseAddresses, dealAddresses, rawMessage, rawAcknowledgment, req.Height)
	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_UPDATED_MESSAGE:
		err = s.finalizeERNUpdate(ctx, req, txhash, ernAddress, senderAddress, ern.Header.Nonce, int16(ern.Header.ControlType), partyAddresses, resourceAddresses, releaseAddresses, dealAddresses, rawMessage, rawAcknowledgment, req.Height)
	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_TAKEDOWN_MESSAGE:
		err = s.finalizeERNTakedown(ctx, req, ernAddress, senderAddress, ern.Header.Nonce, int16(ern.Header.ControlType), partyAddresses, resourceAddresses, releaseAddresses, dealAddresses, rawMessage, req.Height)
	default:
		return fmt.Errorf("unsupported ERN message control type: %v", ern.Header.ControlType)
	}

	return err
}

func (s *Server) finalizeERNCreate(ctx context.Context, _ *abcitypes.FinalizeBlockRequest, txhash string, index int64, address, sender string, nonce uint64, messageControlType int16, partyAddresses, resourceAddresses, releaseAddresses, dealAddresses []string, rawMessage, rawAcknowledgment []byte, blockHeight int64) error {
	qtx := s.getDb()

	// Convert nonce to int64 for database storage
	dbNonce := int64(nonce)

	return qtx.InsertCoreERN(ctx, db.InsertCoreERNParams{
		TxHash:             txhash,
		Index:              index,
		Address:            address,
		Sender:             sender,
		Nonce:              dbNonce,
		MessageControlType: messageControlType,
		PartyAddresses:     partyAddresses,
		ResourceAddresses:  resourceAddresses,
		ReleaseAddresses:   releaseAddresses,
		DealAddresses:      dealAddresses,
		RawMessage:         rawMessage,
		RawAcknowledgment:  rawAcknowledgment,
		BlockHeight:        blockHeight,
	})
}

func (s *Server) finalizeERNUpdate(ctx context.Context, req *abcitypes.FinalizeBlockRequest, txhash string, address, sender string, nonce uint64, messageControlType int16, partyAddresses, resourceAddresses, releaseAddresses, dealAddresses []string, rawMessage, rawAcknowledgment []byte, blockHeight int64) error {
	// TODO: Implement ERN update logic
	return nil
}

func (s *Server) finalizeERNTakedown(ctx context.Context, req *abcitypes.FinalizeBlockRequest, address, sender string, nonce uint64, messageControlType int16, partyAddresses, resourceAddresses, releaseAddresses, dealAddresses []string, rawMessage []byte, blockHeight int64) error {
	// TODO: Implement ERN takedown logic
	return nil
}

func (s *Server) finalizeMEAD(ctx context.Context, req *abcitypes.FinalizeBlockRequest, txhash string, index int64, tx *v1beta1.Transaction, mead *v1beta2.MediaEnrichmentDescription) error {
	// Use envelope nonce as string for address generation
	envelopeNonce := tx.Envelope.Header.Nonce

	meadAddress := common.CreateAddress(mead, s.config.GenesisFile.ChainID, req.Height, envelopeNonce)

	// Marshal the entire MEAD message
	rawMessage, err := proto.Marshal(mead)
	if err != nil {
		return fmt.Errorf("failed to marshal MEAD message: %w", err)
	}

	// TODO: Recover sender address from transaction signature
	senderAddress := mead.Header.From

	// Handle different message control types - use MEAD header nonce for database storage
	switch mead.Header.ControlType {
	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_NEW_MESSAGE:
		err = s.finalizeMEADCreate(ctx, req, txhash, index, meadAddress, senderAddress, mead.Header.Nonce, int16(mead.Header.ControlType), mead.ResourceAddresses, mead.ReleaseAddresses, rawMessage, req.Height)
	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_UPDATED_MESSAGE:
		err = s.finalizeMEADUpdate(ctx, req, meadAddress, senderAddress, mead.Header.Nonce, int16(mead.Header.ControlType), mead.ResourceAddresses, mead.ReleaseAddresses, rawMessage, req.Height)
	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_TAKEDOWN_MESSAGE:
		err = s.finalizeMEADTakedown(ctx, req, meadAddress, senderAddress, mead.Header.Nonce, int16(mead.Header.ControlType), mead.ResourceAddresses, mead.ReleaseAddresses, rawMessage, req.Height)
	default:
		return fmt.Errorf("unsupported MEAD message control type: %v", mead.Header.ControlType)
	}

	return err
}

func (s *Server) finalizePIE(ctx context.Context, req *abcitypes.FinalizeBlockRequest, txhash string, index int64, tx *v1beta1.Transaction, pie *v1beta2.PartyIdentificationEnrichment) error {
	// Use envelope nonce as string for address generation
	envelopeNonce := tx.Envelope.Header.Nonce

	pieAddress := common.CreateAddress(pie, s.config.GenesisFile.ChainID, req.Height, envelopeNonce)

	// Marshal the entire PIE message
	rawMessage, err := proto.Marshal(pie)
	if err != nil {
		return fmt.Errorf("failed to marshal PIE message: %w", err)
	}

	// TODO: Recover sender address from transaction signature
	senderAddress := pie.Header.From

	// Handle different message control types - use PIE header nonce for database storage
	switch pie.Header.ControlType {
	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_NEW_MESSAGE:
		err = s.finalizePIECreate(ctx, req, txhash, index, pieAddress, senderAddress, pie.Header.Nonce, int16(pie.Header.ControlType), pie.PartyAddresses, rawMessage, req.Height)
	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_UPDATED_MESSAGE:
		err = s.finalizePIEUpdate(ctx, req, pieAddress, senderAddress, pie.Header.Nonce, int16(pie.Header.ControlType), pie.PartyAddresses, rawMessage, req.Height)
	case v1beta2.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_TAKEDOWN_MESSAGE:
		err = s.finalizePIETakedown(ctx, req, pieAddress, senderAddress, pie.Header.Nonce, int16(pie.Header.ControlType), pie.PartyAddresses, rawMessage, req.Height)
	default:
		return fmt.Errorf("unsupported PIE message control type: %v", pie.Header.ControlType)
	}

	return err
}

// MEAD helper functions
func (s *Server) finalizeMEADCreate(ctx context.Context, _ *abcitypes.FinalizeBlockRequest, txhash string, index int64, address, sender string, nonce uint64, messageControlType int16, resourceAddresses, releaseAddresses []string, rawMessage []byte, blockHeight int64) error {
	qtx := s.getDb()

	// Convert nonce to int64 for database storage
	dbNonce := int64(nonce)

	return qtx.InsertCoreMEAD(ctx, db.InsertCoreMEADParams{
		TxHash:             txhash,
		Index:              index,
		Address:            address,
		Sender:             sender,
		Nonce:              dbNonce,
		MessageControlType: messageControlType,
		ResourceAddresses:  resourceAddresses,
		ReleaseAddresses:   releaseAddresses,
		RawMessage:         rawMessage,
		BlockHeight:        blockHeight,
	})
}

func (s *Server) finalizeMEADUpdate(ctx context.Context, req *abcitypes.FinalizeBlockRequest, address, sender string, nonce uint64, messageControlType int16, resourceAddresses, releaseAddresses []string, rawMessage []byte, blockHeight int64) error {
	return nil
}

func (s *Server) finalizeMEADTakedown(ctx context.Context, req *abcitypes.FinalizeBlockRequest, address, sender string, nonce uint64, messageControlType int16, resourceAddresses, releaseAddresses []string, rawMessage []byte, blockHeight int64) error {
	return nil
}

// PIE helper functions
func (s *Server) finalizePIECreate(ctx context.Context, _ *abcitypes.FinalizeBlockRequest, txhash string, index int64, address, sender string, nonce uint64, messageControlType int16, partyAddresses []string, rawMessage []byte, blockHeight int64) error {
	qtx := s.getDb()

	// Convert nonce to int64 for database storage
	dbNonce := int64(nonce)

	return qtx.InsertCorePIE(ctx, db.InsertCorePIEParams{
		TxHash:             txhash,
		Index:              index,
		Address:            address,
		Sender:             sender,
		Nonce:              dbNonce,
		MessageControlType: messageControlType,
		PartyAddresses:     partyAddresses,
		RawMessage:         rawMessage,
		BlockHeight:        blockHeight,
	})
}

func (s *Server) finalizePIEUpdate(ctx context.Context, req *abcitypes.FinalizeBlockRequest, address, sender string, nonce uint64, messageControlType int16, partyAddresses []string, rawMessage []byte, blockHeight int64) error {
	return nil
}

func (s *Server) finalizePIETakedown(ctx context.Context, req *abcitypes.FinalizeBlockRequest, address, sender string, nonce uint64, messageControlType int16, partyAddresses []string, rawMessage []byte, blockHeight int64) error {
	return nil
}

// Helper functions to create acknowledgments
func (s *Server) createERNAcknowledgment(ern *v1beta2.ElectronicReleaseNotification, tx *v1beta1.Transaction, blockHeight int64) *v1beta2.ElectronicReleaseNotificationAck {
	envelopeNonce := tx.Envelope.Header.Nonce
	ernAddress := common.CreateAddress(ern, s.config.GenesisFile.ChainID, blockHeight, envelopeNonce)

	// Collect all addresses
	partyAddresses := make([]string, len(ern.PartyList))
	for i, party := range ern.PartyList {
		partyAddresses[i] = common.CreateAddress(party, s.config.GenesisFile.ChainID, blockHeight, envelopeNonce)
	}

	resourceAddresses := make([]string, len(ern.ResourceList))
	for i, resource := range ern.ResourceList {
		resourceAddresses[i] = common.CreateAddress(resource, s.config.GenesisFile.ChainID, blockHeight, envelopeNonce)
	}

	releaseAddresses := make([]string, len(ern.ReleaseList))
	for i, release := range ern.ReleaseList {
		releaseAddresses[i] = common.CreateAddress(release, s.config.GenesisFile.ChainID, blockHeight, envelopeNonce)
	}

	dealAddresses := make([]string, len(ern.DealList))
	for i, deal := range ern.DealList {
		dealAddresses[i] = common.CreateAddress(deal, s.config.GenesisFile.ChainID, blockHeight, envelopeNonce)
	}

	return &v1beta2.ElectronicReleaseNotificationAck{
		ErnAddress:        ernAddress,
		Nonce:             ern.Header.Nonce,
		PartyAddresses:    partyAddresses,
		ResourceAddresses: resourceAddresses,
		ReleaseAddresses:  releaseAddresses,
		DealAddresses:     dealAddresses,
	}
}

func (s *Server) createMEADAcknowledgment(mead *v1beta2.MediaEnrichmentDescription, tx *v1beta1.Transaction, blockHeight int64) *v1beta2.MediaEnrichmentDescriptionAck {
	envelopeNonce := tx.Envelope.Header.Nonce
	meadAddress := common.CreateAddress(mead, s.config.GenesisFile.ChainID, blockHeight, envelopeNonce)

	return &v1beta2.MediaEnrichmentDescriptionAck{
		MeadAddress: meadAddress,
		Nonce:       mead.Header.Nonce,
	}
}

func (s *Server) createPIEAcknowledgment(pie *v1beta2.PartyIdentificationEnrichment, tx *v1beta1.Transaction, blockHeight int64) *v1beta2.PartyIdentificationEnrichmentAck {
	envelopeNonce := tx.Envelope.Header.Nonce
	pieAddress := common.CreateAddress(pie, s.config.GenesisFile.ChainID, blockHeight, envelopeNonce)

	return &v1beta2.PartyIdentificationEnrichmentAck{
		PieAddress: pieAddress,
		Nonce:      pie.Header.Nonce,
	}
}
