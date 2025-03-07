package server

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"

	"github.com/AudiusProject/audiusd/pkg/core/common"
	"github.com/AudiusProject/audiusd/pkg/core/db"
	"github.com/AudiusProject/audiusd/pkg/core/gen/core_proto"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	cometcrypto "github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cometbft/cometbft/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/proto"
)

const maxRegistrationAttestationValidity = 24 * 60 * 60 // Registration requests are only valid for approx 24 hours

func (s *Server) isValidRegisterNodeAttestation(ctx context.Context, tx *core_proto.SignedTransaction, signers []string, blockHeight int64) error {
	vr := tx.GetAttestation().GetValidatorRegistration()
	if vr == nil {
		return fmt.Errorf("unknown tx fell into isValidRegisterNodeAttestation: %v", tx)
	}

	// validate address from tx signature
	sig := tx.GetSignature()
	if sig == "" {
		return fmt.Errorf("no signature provided for registration tx: %v", tx)
	}
	attBytes, err := proto.Marshal(tx.GetAttestation())
	if err != nil {
		return fmt.Errorf("could not marshal registration tx: %v", err)
	}
	_, address, err := common.EthRecover(tx.GetSignature(), attBytes)
	if err != nil {
		return fmt.Errorf("could not recover msg sig: %v", err)
	}
	if address != vr.GetDelegateWallet() {
		return fmt.Errorf("signature address '%s' does not match ethereum registration '%s'", address, vr.GetDelegateWallet())
	}

	// validate voting power
	if vr.GetPower() != int64(s.config.ValidatorVotingPower) {
		return fmt.Errorf("invalid voting power '%d'", vr.GetPower())
	}

	// validate pub key
	if len(vr.GetPubKey()) == 0 {
		return fmt.Errorf("public Key missing from %s registration tx", vr.GetEndpoint())
	}
	vrPubKey := ed25519.PubKey(vr.GetPubKey())
	if vrPubKey.Address().String() != vr.GetCometAddress() {
		return fmt.Errorf("address does not match public key: %s %s", vrPubKey.Address(), vr.GetCometAddress())
	}

	// ensure comet address is not already taken
	if _, err := s.db.GetRegisteredNodeByCometAddress(context.Background(), vr.GetCometAddress()); !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("address '%s' is already registered on comet, node %s attempted to acquire it", vr.GetCometAddress(), vr.GetEndpoint())
	}

	// validate age of request
	if vr.Deadline < blockHeight || vr.Deadline > blockHeight+maxRegistrationAttestationValidity {
		return fmt.Errorf("Registration request for '%s' with deadline %d is too new/old (current height is %d)", vr.GetEndpoint(), vr.Deadline, blockHeight)
	}

	// validate signers
	keyBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(keyBytes, uint64(vr.GetEthBlock()))
	enough, err := s.attestationHasEnoughSigners(ctx, signers, keyBytes, s.config.AttRegistrationRSize, s.config.AttRegistrationMin)
	if err != nil {
		return fmt.Errorf("Error checking attestors against validators: %v", err)
	} else if !enough {
		return fmt.Errorf("Not enough attestations provided to register validator at '%s'", vr.GetEndpoint())
	}

	return nil
}

func (s *Server) finalizeRegisterNodeAttestation(ctx context.Context, tx *core_proto.SignedTransaction, blockHeight int64) error {
	qtx := s.getDb()
	vr := tx.GetAttestation().GetValidatorRegistration()

	txBytes, err := proto.Marshal(tx)
	if err != nil {
		return fmt.Errorf("could not unmarshal tx bytes: %v", err)
	}
	pubKey, _, err := common.EthRecover(tx.GetSignature(), txBytes)
	if err != nil {
		return fmt.Errorf("could not recover signer: %v", err)
	}

	serializedPubKey, err := common.SerializePublicKey(pubKey)
	if err != nil {
		return fmt.Errorf("could not serialize pubkey: %v", err)
	}

	// Do not reinsert duplicate registrations
	if _, err = qtx.GetRegisteredNodeByEthAddress(ctx, vr.GetDelegateWallet()); errors.Is(err, pgx.ErrNoRows) {
		err = qtx.InsertRegisteredNode(ctx, db.InsertRegisteredNodeParams{
			PubKey:       serializedPubKey,
			EthAddress:   vr.GetDelegateWallet(),
			Endpoint:     vr.GetEndpoint(),
			CometAddress: vr.GetCometAddress(),
			CometPubKey:  base64.StdEncoding.EncodeToString(vr.GetPubKey()),
			EthBlock:     strconv.FormatInt(vr.GetEthBlock(), 10),
			NodeType:     vr.GetNodeType(),
			SpID:         vr.GetSpId(),
		})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("error inserting registered node: %v", err)
		}
	}
	return nil
}

func (s *Server) isValidDeregisterNodeTx(tx *core_proto.SignedTransaction, misbehavior []abcitypes.Misbehavior) error {
	sig := tx.GetSignature()
	if sig == "" {
		return fmt.Errorf("no signature provided for deregistration tx: %v", tx)
	}

	vd := tx.GetValidatorDeregistration()
	if vd == nil {
		return fmt.Errorf("unknown tx fell into isValidDeregisterNodeTx: %v", tx)
	}

	addr := vd.GetCometAddress()

	_, err := s.db.GetRegisteredNodeByCometAddress(context.Background(), addr)
	if err != nil {
		return fmt.Errorf("not able to find registered node: %v", err)
	}

	if len(vd.GetPubKey()) == 0 {
		return fmt.Errorf("public Key missing from deregistration tx: %v", tx)
	}
	vdPubKey := ed25519.PubKey(vd.GetPubKey())
	if vdPubKey.Address().String() != addr {
		return fmt.Errorf("address does not match public key: %s %s", vdPubKey.Address(), addr)
	}

	for _, mb := range misbehavior {
		validator := mb.GetValidator()
		if addr == cometcrypto.Address(validator.GetAddress()).String() {
			return nil
		}
	}

	return fmt.Errorf("no misbehavior found matching deregistration tx: %v", tx)
}

func (s *Server) finalizeDeregisterNode(ctx context.Context, tx *core_proto.SignedTransaction, misbehavior []abcitypes.Misbehavior) (*core_proto.ValidatorDeregistration, error) {
	if err := s.isValidDeregisterNodeTx(tx, misbehavior); err != nil {
		return nil, fmt.Errorf("invalid deregister node tx: %v", err)
	}

	vd := tx.GetValidatorDeregistration()
	qtx := s.getDb()
	err := qtx.DeleteRegisteredNode(ctx, vd.GetCometAddress())
	if err != nil {
		return nil, fmt.Errorf("error deleting registered node: %v", err)
	}

	return vd, nil
}

func (s *Server) createDeregisterTransaction(address types.Address) ([]byte, error) {
	node, err := s.db.GetRegisteredNodeByCometAddress(context.Background(), address.String())
	if err != nil {
		return []byte{}, fmt.Errorf("not able to find registered node with address '%s': %v", address.String(), err)
	}
	pubkeyEnc, err := base64.StdEncoding.DecodeString(node.CometPubKey)
	if err != nil {
		return []byte{}, fmt.Errorf("could not decode public key '%s' as base64 encoded string: %v", node.CometPubKey, err)
	}
	deregistrationTx := &core_proto.ValidatorDeregistration{
		PubKey:       pubkeyEnc,
		CometAddress: address.String(),
	}

	txBytes, err := proto.Marshal(deregistrationTx)
	if err != nil {
		return []byte{}, fmt.Errorf("failure to marshal deregister tx: %v", err)
	}

	sig, err := common.EthSign(s.config.EthereumKey, txBytes)
	if err != nil {
		return []byte{}, fmt.Errorf("could not sign deregister tx: %v", err)
	}

	tx := core_proto.SignedTransaction{
		Signature: sig,
		RequestId: uuid.NewString(),
		Transaction: &core_proto.SignedTransaction_ValidatorDeregistration{
			ValidatorDeregistration: deregistrationTx,
		},
	}

	signedTxBytes, err := proto.Marshal(&tx)
	if err != nil {
		return []byte{}, err
	}
	return signedTxBytes, nil
}
