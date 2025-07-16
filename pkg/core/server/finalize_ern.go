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

			switch ern.MessageHeader.MessageControlType {
			case v1beta2.MessageControlType_MESSAGE_CONTROL_TYPE_NEW_RELEASE_MESSAGE:
				err = s.finalizeERNCreate(ctx, req, tx, ern)
			case v1beta2.MessageControlType_MESSAGE_CONTROL_TYPE_UPDATED_RELEASE_MESSAGE:
				err = s.finalizeERNUpdate()
			case v1beta2.MessageControlType_MESSAGE_CONTROL_TYPE_TAKEDOWN_RELEASE_MESSAGE:
				err = s.finalizeERNTakeDown()
			}
		}

		if err != nil {
			return fmt.Errorf("failed to finalize ERN: %w", err)
		}
	}

	if err != nil {
		return err
	}

	return nil
}

func (s *Server) finalizeERNCreate(ctx context.Context, req *abcitypes.FinalizeBlockRequest, tx *v1beta1.Transaction, ern *v1beta2.NewReleaseMessage) error {
	txHash := s.toTxHash(tx.Envelope)
	ernAddress := common.CreateAddress(ern, s.config.GenesisFile.ChainID, req.Height, tx.Envelope.Header.Nonce)

	releaseAddresses := make([]string, len(ern.ReleaseList))
	soundRecordingAddresses := make([]string, len(ern.ResourceList))

	for i, release := range ern.ReleaseList {
		releaseAddresses[i] = common.CreateAddress(release, s.config.GenesisFile.ChainID, req.Height, tx.Envelope.Header.Nonce)
	}

	for i, resource := range ern.ResourceList {
		soundRecordingAddresses[i] = common.CreateAddress(resource, s.config.GenesisFile.ChainID, req.Height, tx.Envelope.Header.Nonce)
	}

	rawErnMessage, err := proto.Marshal(ern)
	if err != nil {
		return fmt.Errorf("failed to marshal ERN message: %w", err)
	}

	// TODO: recover sender address from tx
	senderAddress := ""

	qtx := s.getDb()

	qtx.InsertERNMessage(ctx, db.InsertERNMessageParams{
		Address:       ernAddress,
		TxHash:        txHash,
		BlockHeight:   req.Height,
		SenderAddress: senderAddress,
		RawErnMessage: rawErnMessage,
	})

	qtx.InsertERNReleaseAddresses(ctx, db.InsertERNReleaseAddressesParams{
		Column1:    releaseAddresses,
		ErnAddress: ernAddress,
	})

	qtx.InsertERNSoundRecordingAddresses(ctx, db.InsertERNSoundRecordingAddressesParams{
		Column1:    soundRecordingAddresses,
		ErnAddress: ernAddress,
	})

	// TODO: persist ERN to storage
	return nil
}

func (s *Server) finalizeERNUpdate() error { return nil }

func (s *Server) finalizeERNTakeDown() error { return nil }
