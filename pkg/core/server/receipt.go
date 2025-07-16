package server

import (
	"context"

	"github.com/AudiusProject/audiusd/pkg/api/core/v1beta1"
)

func (s *CoreService) buildTxReceipt(ctx context.Context, incomingTx *v1beta1.Transaction) (*v1beta1.TransactionReceipt, error) {
	receipt := new(v1beta1.TransactionReceipt)

	return receipt, nil
}
