package server

import (
	"context"

	"github.com/AudiusProject/audiusd/pkg/core/gen/core_proto"
)

func (s *Server) isValidReleaseTx(ctx context.Context, tx *core_proto.SignedTransaction) error {
	return nil
}

func (s *Server) finalizeRelease(ctx context.Context, tx *core_proto.SignedTransaction, txHash string) (*core_proto.SignedTransaction, error) {
	return tx, nil
}
