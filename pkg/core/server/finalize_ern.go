package server

import (
	"context"
	"fmt"

	"github.com/AudiusProject/audiusd/pkg/api/core/v1beta1"
	"github.com/AudiusProject/audiusd/pkg/api/ddex/v1beta2"
	"github.com/AudiusProject/audiusd/pkg/common"
	abcitypes "github.com/cometbft/cometbft/abci/types"
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
	ernAddress := common.CreateAddress(ern, s.config.GenesisFile.ChainID, req.Height, tx.Envelope.Header.Nonce)
	s.logger.Info("finalizing ERN create", "ern_address", ernAddress)
	// TODO: persist ERN to storage
	return nil
}

func (s *Server) finalizeERNUpdate() error { return nil }

func (s *Server) finalizeERNTakeDown() error { return nil }
