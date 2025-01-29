package server

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/AudiusProject/audiusd/pkg/core/common"
	"github.com/AudiusProject/audiusd/pkg/core/db"
	"github.com/AudiusProject/audiusd/pkg/core/gen/core_proto"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/proto"
)

func (s *Server) submitStorageProofTx(height int64, hash []byte, cid string, replicasAddresses []string, proof []byte, verifierAddress string, verifierPubKey string) error {
	secret, err := generateRandomSecret()
	if err != nil {
		return fmt.Errorf("Could not generate random secret: %v", err)
	}
	encProof, err := encryptStorageProof(secret, proof, hash)
	if err != nil {
		return fmt.Errorf("Could not encrypt storage proof: %v", err)
	}
	pubKeyBytes, err := base64.StdEncoding.DecodeString(verifierPubKey)
	if err != nil {
		return fmt.Errorf("Could not decode public key: %v", err)
	}
	proofSig, err := s.config.CometKey.Sign(proof)
	if err != nil {
		return fmt.Errorf("Could not sign storage proof: %v", err)
	}
	proofTx := &core_proto.StorageProof{
		Height:          height,
		Cid:             cid,
		Address:         s.config.ProposerAddress,
		ProofSignature:  proofSig,
		ProverAddresses: replicas,
	}

	txBytes, err := proto.Marshal(proofTx)
	if err != nil {
		return fmt.Errorf("failure to marshal proof tx: %v", err)
	}

	sig, err := common.EthSign(s.config.EthereumKey, txBytes)
	if err != nil {
		return fmt.Errorf("could not sign proof tx: %v", err)
	}

	tx := &core_proto.SignedTransaction{
		Signature: sig,
		RequestId: uuid.NewString(),
		Transaction: &core_proto.SignedTransaction_StorageProof{
			StorageProof: proofTx,
		},
	}

	req := &core_proto.SendTransactionRequest{
		Transaction: tx,
	}

	txhash, err := s.SendTransaction(context.Background(), req)
	if err != nil {
		return fmt.Errorf("send storage proof tx failed: %v", err)
	}
	s.logger.Infof("Sent storage proof for cid '%s' at height '%d', receipt '%s'", cid, height, txhash)

	// Send the verification later.
	go func() {
		time.Sleep(posVerificationDelay)
		s.submitStorageProofVerificationTx(proof)
	}()

	return nil
}

func (s *Server) submitStorageProofVerificationTx(proof []byte) error {
	verificationTx := &core_proto.StorageProofVerification{
		Height: height,
		Proof:  proof,
	}

	txBytes, err := proto.Marshal(verificationTx)
	if err != nil {
		return fmt.Errorf("failure to marshal proof tx: %v", err)
	}

	sig, err := common.EthSign(s.config.EthereumKey, txBytes)
	if err != nil {
		return fmt.Errorf("could not sign proof tx: %v", err)
	}

	tx := &core_proto.SignedTransaction{
		Signature: sig,
		RequestId: uuid.NewString(),
		Transaction: &core_proto.SignedTransaction_StorageProofVerification{
			StorageProofVerification: verificationTx,
		},
	}

	req := &core_proto.SendTransactionRequest{
		Transaction: tx,
	}

	txhash, err := s.SendTransaction(context.Background(), req)
	if err != nil {
		return fmt.Errorf("send storage proof verification tx failed: %v", err)
	}
	s.logger.Infof("Sent storage proof verification for pos challenge at height '%d', receipt '%s'", height, txhash)
	return nil
}

func (s *Server) isValidStorageProofTx(ctx context.Context, tx *core_proto.SignedTransaction, currentBlockHeight int64, enforceReplicas bool) error {
	// validate signer == prover
	sig := tx.GetSignature()
	if sig == "" {
		return fmt.Errorf("no signature provided for storage proof tx: %v", tx)
	}
	sp := tx.GetStorageProof()
	if sp == nil {
		return fmt.Errorf("unknown tx fell into isValidStorageProofTx: %v", tx)
	}
	txBytes, err := proto.Marshal(sp)
	if err != nil {
		return fmt.Errorf("could not unmarshal tx bytes: %v", err)
	}
	_, address, err := common.EthRecover(sig, txBytes)
	if err != nil {
		return nil, fmt.Errorf("could not recover signer: %v", err)
	}
	node, err := s.db.GetRegisteredNodeByEthAddress(ctx, address)
	if err != nil {
		return fmt.Errorf("Could not get validator for address '%s': %v", err)
	}
	if strings.ToLower(node.CometAddress) != strings.ToLower(sp.Address) {
		return fmt.Errorf("Proof is for '%s' but was signed by '%s'", sp.Address, node.CometAddress)
	}

	// validate height
	height := sp.GetHeight()
	if height == 0 {
		return fmt.Errorf("Invalid height '%d' for storage proof", height)
	}
	if height-currentBlockHeight > posChallengeDeadline {
		return fmt.Errorf("Proof submitted at height '%d' for challenge at height '%d' which is past the deadline", currentBlockHeight, height)
	}

	// validate existing ongoing challenge
	posChallenge, err := s.db.GetPoSChallenge(ctx, height)
	if err != nil {
		return fmt.Errorf("Could not retrieve any ongoing pos challenges at height '%d': %v", height, err)
	}
	if enforceReplicas && posChallenge.ProverAddresses != nil && !slices.Contains(posChallenge.ProverAddresses, strings.ToLower(sp.Address)) {
		// We think this proof does not belong to this challenge.
		// Note: this should not be enforced during the finalize step.
		return fmt.Errorf("Prover at address '%s' does not belong to replicaset.", sp.Address)
	}

	return nil
}

func (s *Server) isValidStorageProofVerificationTx(ctx context.Context, tx *core_proto.SignedTransaction, currentBlockHeight int64) error {
	spv := tx.GetStorageProofVerification()
	if spv == nil {
		return fmt.Errorf("unknown tx fell into isValidStorageProofVerficationTx: %v", tx)
	}

	// validate height
	height := spv.GetHeight()
	if height == 0 {
		return fmt.Errorf("Invalid height '%d' for storage proof", height)
	}
	if height-currentBlockHeight <= posChallengeDeadline {
		return fmt.Errorf("Proof submitted at height '%d' for challenge at height '%d' which is before the deadline", currentBlockHeight, height)
	}

	// validate against existing proof
	storageProof, err := s.db.GetStorageProof(
		ctx,
		db.GetStorageProofParams{height, spv.Address},
	)
	if err != nil {
		return fmt.Errorf("Could not retrieve any existing storage proof for node '%s' at height '%d': %v", spv.Address, height, err)
	}
	pubKeyBytes, err := base64.StdEncoding.DecodeString(node.CometPubKey)
	if err != nil {
		return fmt.Errorf("Could not decode public key for node at address %s: %v", spv.Address, err)
	}
	pubKey := ed25519.PubKey(pubKeyBytes)
	if !pubKey.VerifySignature(spv.Proof, storageProof.ProofSignature) {
		return fmt.Errorf("Signature in storage proof for node %s at block %d does not match proof", spv.Address, height)
	}

	return nil
}

func (s *Server) finalizeStorageProof(ctx context.Context, tx *core_proto.SignedTransaction, blockHeight int64) (*core_proto.StorageProof, error) {
	if err := s.isValidStorageProofTx(ctx, tx, blockHeight, false); err != nil {
		return nil, err
	}

	sp := tx.GetStorageProof()
	qtx := s.getDb()

	// ignore duplicates
	if _, err := qtx.GetStorageProof(ctx, db.GetStorageProofParams{sp.Height, sp.Address}); !errors.Is(err, pgx.ErrNoRows) {
		s.logger.Error("Storage proof already exists, skipping.", "node", sp.Address, "height", sp.Height)
		return sp, nil
	}

	proofSigStr := base64.StdEncoding.EncodeToString(sp.ProofSignature)

	if err := qtx.InsertStorageProof(
		ctx,
		db.InsertStorageProofParams{
			BlockHeight:     sp.Height,
			Address:         sp.Address,
			Cid:             pgtype.Text{sp.Cid, true},
			ProofSignature:  pgtype.Text{proofSigStr, true},
			ProverAddresses: sp.ProverAddresses,
		},
	); err != nil {
		return nil, fmt.Errorf("Could not persist storage proof in db: %v", err)
	}

	return sp, nil
}

func (s *Server) finalizeStorageProofVerification(ctx context.Context, tx *core_proto.SignedTransaction, blockHeight int64) (*core_proto.StorageProof, error) {
	if err := s.isValidStorageProofVerificationTx(ctx, tx, blockHeight); err != nil {
		return nil, err
	}

	spv := tx.GetStorageProofVerification()
	qtx := s.getDb()

	proofs, err := qtx.GetStorageProofs(ctx, spv.Height)
	if err != nil {
		return nil, fmt.Errorf("Could not fetch storage proofs: %v", err)
	}

	consensusProofs := make([]db.StorageProof, 0, len(proofs))
	for _, p := range proofs {
		node, err := qtx.GetRegisteredNodeByCometAddress(ctx, p.Address)
		if err != nil {
			return nil, fmt.Errorf("Could not fetch node with address %s: %v", p.Address, err)
		}

		sigBytes, err := base64.StdEncoding.DecodeString(p.ProofSignature.String)
		if err != nil {
			return fmt.Errorf("Could not decode proof signature node at address %s: %v", node.CometAddress, err)
		}
		pubKeyBytes, err := base64.StdEncoding.DecodeString(node.CometPubKey)
		if err != nil {
			return fmt.Errorf("Could not decode public key for node at address %s: %v", node.CometAddress, err)
		}
		pubKey := ed25519.PubKey(pubKeyBytes)
		if pubKey.VerifySignature(spv.Proof, sigBytes) {
			consensusProofs = append(consensusProofs, p)
		}
	}

	if len(consensusProofs) > len(proofs)/2 {
		// TODO
		// we have a majority, we can resolve the challenge
	}

	return sp, nil
}
