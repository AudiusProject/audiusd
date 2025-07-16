package server

import (
	"context"
	"fmt"

	"github.com/AudiusProject/audiusd/pkg/api/core/v1beta1"
	"github.com/AudiusProject/audiusd/pkg/common"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	"google.golang.org/protobuf/proto"
)

func (s *Server) finalizeV2Transaction(ctx context.Context, req *abcitypes.FinalizeBlockRequest, tx *v1beta1.Transaction) (proto.Message, error) {
	header := tx.Envelope.Header
	if header.ChainId != s.config.GenesisFile.ChainID {
		return nil, fmt.Errorf("invalid chain id: %s", header.ChainId)
	}

	if header.Expiration < req.Height {
		return nil, fmt.Errorf("transaction expired")
	}

	var err error
	for _, msg := range tx.Envelope.Messages {
		switch msg.Message.(type) {
		case *v1beta1.Message_Ern:
			ern := msg.GetErn()

			ernAddress := common.CreateAddress(ern, s.config.GenesisFile.ChainID, req.Height, tx.Envelope.Header.Nonce)

			// TODO: finalize ERN
		}
	}

	if err != nil {
		return nil, err
	}

	return tx, nil
}
