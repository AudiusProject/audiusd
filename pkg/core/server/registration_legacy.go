package server

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/AudiusProject/audiusd/pkg/core/common"
	"github.com/AudiusProject/audiusd/pkg/core/db"
	"github.com/AudiusProject/audiusd/pkg/core/gen/core_proto"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/proto"
)

// checks if the register node tx is valid
// calls ethereum mainnet and validates signature to confirm node should be a validator
func (s *Server) isValidLegacyRegisterNodeTx(tx *core_proto.SignedTransaction) error {
	sig := tx.GetSignature()
	if sig == "" {
		return fmt.Errorf("no signature provided for registration tx: %v", tx)
	}

	vr := tx.GetValidatorRegistrationLegacy()
	if vr == nil {
		return fmt.Errorf("unknown tx fell into isValidLegacyRegisterNodeTx: %v", tx)
	}

	info, err := s.getRegisteredNode(vr.GetEndpoint())
	if err != nil {
		return fmt.Errorf("not able to find registered node: %v", err)
	}

	// compare on chain info to requested comet data
	onChainOwnerWallet := info.DelegateOwnerWallet.Hex()
	onChainBlockNumber := info.BlockNumber.String()
	onChainEndpoint := info.Endpoint

	if err := s.isDuplicateDelegateOwnerWallet(onChainOwnerWallet); err != nil {
		return err
	}

	data, err := proto.Marshal(vr)
	if err != nil {
		return fmt.Errorf("could not marshal registration tx: %v", err)
	}

	_, address, err := common.EthRecover(tx.GetSignature(), data)
	if err != nil {
		return fmt.Errorf("could not recover msg sig: %v", err)
	}

	vrOwnerWallet := address
	vrEndpoint := vr.GetEndpoint()
	vrEthBlock := vr.GetEthBlock()
	vrCometAddress := vr.GetCometAddress()
	vrPower := int(vr.GetPower())

	if len(vr.GetPubKey()) == 0 {
		return fmt.Errorf("public Key missing from %s registration tx", vrEndpoint)
	}
	vrPubKey := ed25519.PubKey(vr.GetPubKey())

	if onChainOwnerWallet != vrOwnerWallet {
		return fmt.Errorf("wallet %s tried to register %s as %s", vrOwnerWallet, onChainOwnerWallet, vr.Endpoint)
	}

	if onChainBlockNumber != vrEthBlock {
		return fmt.Errorf("block number mismatch: %s %s", onChainBlockNumber, vrEthBlock)
	}

	if onChainEndpoint != vrEndpoint {
		return fmt.Errorf("endpoints don't match: %s %s", onChainEndpoint, vrEndpoint)
	}

	if vrPubKey.Address().String() != vrCometAddress {
		return fmt.Errorf("address does not match public key: %s %s", vrPubKey.Address(), vrCometAddress)
	}

	if vrPower != s.config.ValidatorVotingPower {
		return fmt.Errorf("invalid voting power '%d'", vrPower)
	}

	return nil
}

// persists the register node request should it pass validation
func (s *Server) finalizeLegacyRegisterNode(ctx context.Context, tx *core_proto.SignedTransaction, blockTime time.Time) (*core_proto.ValidatorRegistrationLegacy, error) {
	// TODO: remove logic after validator registration switches to attestations
	oldBlock := time.Since(blockTime) >= week
	if !oldBlock {
		if err := s.isValidLegacyRegisterNodeTx(tx); err != nil {
			return nil, fmt.Errorf("invalid register node tx: %v", err)
		}
	}

	qtx := s.getDb()

	vr := tx.GetValidatorRegistrationLegacy()
	sig := tx.GetSignature()
	txBytes, err := proto.Marshal(vr)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal tx bytes: %v", err)
	}

	pubKey, address, err := common.EthRecover(sig, txBytes)
	if err != nil {
		return nil, fmt.Errorf("could not recover signer: %v", err)
	}

	serializedPubKey, err := common.SerializePublicKey(pubKey)
	if err != nil {
		return nil, fmt.Errorf("could not serialize pubkey: %v", err)
	}

	registerNode := tx.GetValidatorRegistrationLegacy()

	// Do not reinsert duplicate registrations
	if _, err = qtx.GetRegisteredNodeByEthAddress(ctx, address); errors.Is(err, pgx.ErrNoRows) {
		err = qtx.InsertRegisteredNode(ctx, db.InsertRegisteredNodeParams{
			PubKey:       serializedPubKey,
			EthAddress:   address,
			Endpoint:     registerNode.GetEndpoint(),
			CometAddress: registerNode.GetCometAddress(),
			CometPubKey:  base64.StdEncoding.EncodeToString(registerNode.GetPubKey()),
			EthBlock:     registerNode.GetEthBlock(),
			NodeType:     registerNode.GetNodeType(),
			SpID:         registerNode.GetSpId(),
		})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("error inserting registered node: %v", err)
		}
	}

	return vr, nil
}
