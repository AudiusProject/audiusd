package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	v1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	"github.com/AudiusProject/audiusd/pkg/api/core/v1/v1connect"
	"github.com/AudiusProject/audiusd/pkg/core/gen/core_proto"
	"google.golang.org/protobuf/proto"
)

type CoreService struct {
	core *Server
}

func NewCoreService() *CoreService {
	return &CoreService{}
}

func (c *CoreService) SetCore(core *Server) {
	c.core = core
}

// ForwardTransaction implements v1connect.CoreServiceHandler.
func (c *CoreService) ForwardTransaction(ctx context.Context, req *connect.Request[v1.ForwardTransactionRequest]) (*connect.Response[v1.ForwardTransactionResponse], error) {
	tx, err := convertV1TransactionToSignedTransaction(req.Msg.Transaction)
	if err != nil {
		return nil, fmt.Errorf("failed to convert transaction: %w", err)
	}
	_, err = c.core.ForwardTransaction(ctx, &core_proto.ForwardTransactionRequest{
		Transaction: tx,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to forward transaction: %w", err)
	}
	return connect.NewResponse(&v1.ForwardTransactionResponse{}), nil
}

// GetBlock implements v1connect.CoreServiceHandler.
func (c *CoreService) GetBlock(ctx context.Context, req *connect.Request[v1.GetBlockRequest]) (*connect.Response[v1.GetBlockResponse], error) {
	block, err := c.core.GetBlock(ctx, &core_proto.GetBlockRequest{Height: req.Msg.Height})
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	transactions := make([]*v1.Transaction, len(block.TransactionResponses))
	for i, tx := range block.TransactionResponses {
		signedTx, err := convertSignedTransactionToV1Transaction(tx.Transaction)
		if err != nil {
			return nil, fmt.Errorf("failed to convert transaction: %w", err)
		}

		transactions[i] = &v1.Transaction{
			Hash:        tx.Txhash,
			BlockHash:   tx.BlockHash,
			ChainId:     block.Chainid,
			Height:      block.Height,
			Timestamp:   block.Timestamp,
			Transaction: signedTx,
		}
	}

	return connect.NewResponse(&v1.GetBlockResponse{Block: &v1.Block{
		Height:       block.Height,
		Hash:         block.Blockhash,
		ChainId:      block.Chainid,
		Proposer:     block.Proposer,
		Timestamp:    block.Timestamp,
		Transactions: transactions,
	}}), nil
}

// GetDeregistrationAttestation implements v1connect.CoreServiceHandler.
func (c *CoreService) GetDeregistrationAttestation(ctx context.Context, req *connect.Request[v1.GetDeregistrationAttestationRequest]) (*connect.Response[v1.GetDeregistrationAttestationResponse], error) {
	deregistrationRequest := req.Msg.Deregistration
	res, err := c.core.GetDeregistrationAttestation(ctx, &core_proto.DeregistrationAttestationRequest{
		Deregistration: &core_proto.ValidatorDeregistration{
			CometAddress: deregistrationRequest.CometAddress,
			PubKey:       deregistrationRequest.PubKey,
			Deadline:     deregistrationRequest.Deadline,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get deregistration attestation: %w", err)
	}
	return connect.NewResponse(&v1.GetDeregistrationAttestationResponse{
		Signature: res.Signature,
		Deregistration: &v1.ValidatorDeregistration{
			CometAddress: res.Deregistration.CometAddress,
			PubKey:       res.Deregistration.PubKey,
			Deadline:     res.Deregistration.Deadline,
		},
	}), nil
}

// GetHealth implements v1connect.CoreServiceHandler.
func (c *CoreService) GetHealth(context.Context, *connect.Request[v1.GetHealthRequest]) (*connect.Response[v1.GetHealthResponse], error) {
	return connect.NewResponse(&v1.GetHealthResponse{}), nil
}

// GetRegistrationAttestation implements v1connect.CoreServiceHandler.
func (c *CoreService) GetRegistrationAttestation(ctx context.Context, req *connect.Request[v1.GetRegistrationAttestationRequest]) (*connect.Response[v1.GetRegistrationAttestationResponse], error) {
	registrationRequest := req.Msg.Registration
	res, err := c.core.GetRegistrationAttestation(ctx, &core_proto.RegistrationAttestationRequest{
		Registration: &core_proto.ValidatorRegistration{
			DelegateWallet: registrationRequest.DelegateWallet,
			CometAddress:   registrationRequest.CometAddress,
			PubKey:         registrationRequest.PubKey,
			Deadline:       registrationRequest.Deadline,
			Endpoint:       registrationRequest.Endpoint,
			NodeType:       registrationRequest.NodeType,
			SpId:           registrationRequest.SpId,
			EthBlock:       registrationRequest.EthBlock,
			Power:          registrationRequest.Power,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get registration attestation: %w", err)
	}
	return connect.NewResponse(&v1.GetRegistrationAttestationResponse{
		Signature: res.Signature,
		Registration: &v1.ValidatorRegistration{
			DelegateWallet: res.Registration.DelegateWallet,
			CometAddress:   res.Registration.CometAddress,
			PubKey:         res.Registration.PubKey,
			Deadline:       res.Registration.Deadline,
			Endpoint:       res.Registration.Endpoint,
			NodeType:       res.Registration.NodeType,
			SpId:           res.Registration.SpId,
			EthBlock:       res.Registration.EthBlock,
			Power:          res.Registration.Power,
		},
	}), nil
}

// GetTransaction implements v1connect.CoreServiceHandler.
func (c *CoreService) GetTransaction(ctx context.Context, req *connect.Request[v1.GetTransactionRequest]) (*connect.Response[v1.GetTransactionResponse], error) {
	tx, err := c.core.GetTransaction(ctx, &core_proto.GetTransactionRequest{Txhash: req.Msg.TxHash})
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	block, err := c.core.GetBlock(ctx, &core_proto.GetBlockRequest{Height: tx.BlockHeight})
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	signedTx, err := convertSignedTransactionToV1Transaction(tx.Transaction)
	if err != nil {
		return nil, fmt.Errorf("failed to convert transaction: %w", err)
	}
	return connect.NewResponse(&v1.GetTransactionResponse{Transaction: &v1.Transaction{
		Hash:        tx.Txhash,
		BlockHash:   tx.BlockHash,
		ChainId:     block.Chainid,
		Height:      block.Height,
		Timestamp:   block.Timestamp,
		Transaction: signedTx,
	}}), nil
}

// Ping implements v1connect.CoreServiceHandler.
func (c *CoreService) Ping(context.Context, *connect.Request[v1.PingRequest]) (*connect.Response[v1.PingResponse], error) {
	return connect.NewResponse(&v1.PingResponse{Message: "pong"}), nil
}

// SendTransaction implements v1connect.CoreServiceHandler.
func (c *CoreService) SendTransaction(context.Context, *connect.Request[v1.SendTransactionRequest]) (*connect.Response[v1.SendTransactionResponse], error) {
	panic("unimplemented")
}

func convertSignedTransactionToV1Transaction(tx *core_proto.SignedTransaction) (*v1.SignedTransaction, error) {
	txBytes, err := proto.Marshal(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction: %w", err)
	}
	signedTx := &v1.SignedTransaction{}
	err = proto.Unmarshal(txBytes, signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
	}
	return signedTx, nil
}

func convertV1TransactionToSignedTransaction(tx *v1.SignedTransaction) (*core_proto.SignedTransaction, error) {
	txBytes, err := proto.Marshal(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction: %w", err)
	}
	signedTx := &core_proto.SignedTransaction{}
	err = proto.Unmarshal(txBytes, signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
	}
	return signedTx, nil
}

var _ v1connect.CoreServiceHandler = (*CoreService)(nil)
