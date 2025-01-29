package server

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/AudiusProject/audiusd/pkg/core/db"
	"github.com/AudiusProject/audiusd/pkg/pos"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
)

const (
	mediorumPoSRequestTimeout = 1 * time.Second
	posChallengeDeadline      = 2
	posVerificationDelay      = posChallengeDeadline * 3 * time.Second
)

// Called during FinalizeBlock. Keeps Proof of Storage subsystem up to date with current block.
func (s *Server) syncPoS(latestBlockHash []byte, latestBlockHeight int64) error {
	if blockShouldTriggerNewPoSChallenge(latestBlockHash) {
		if err := s.triggerNewPoSChallenge(latestBlockHash, latestBlockHeight); err != nil {
			return err
		}
	}
	if err := s.updateExistingPoSChallenges(latestBlockHeight); err != nil {
		return err
	}
	return nil
}

func blockShouldTriggerNewPoSChallenge(blockHash []byte) bool {
	bhLen := len(blockHash)
	return bhLen > 0 && blockHash[bhLen-1]&0x0f == 0
}

func (s *Server) triggerNewPoSChallenge(blockHash []byte, blockHeight int64) error {
	verifier, err := s.getPoSVerifierForChallenge(blockHash)
	if err != nil {
		return fmt.Errorf("Could not get verifier for PoS challenge at height %d with hash %v: %v", blockHeight, blockHash, err)
	}
	err = s.db.InsertPoSChallenge(
		context.Background(),
		db.InsertPoSChallengeParams{blockHeight, verifier.CometAddress},
	)
	if err != nil {
		return fmt.Errorf("Could not insert PoS challenge to db at height %d: %v", blockHeight, err)
	}

	// TODO: disable in block sync mode
	go s.sendPoSChallengeToMediorum(blockHash, blockHeight, verifier.CometAddress, verifier.CometPubKey)
	return nil
}

func (s *Server) updateExistingPoSChallenges(blockHeight int64) error {
	ctx := context.Background()
	openChallenges, err := s.db.GetIncompletePoSChallenges(ctx)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("Could not fetch incomplete pos challenges from the database: %v", err)
	}

	for _, c := range openChallenges {
		if blockHeight-c.BlockHeight >= int64(posChallengeDeadline) && strings.ToLower(c.VerifierAddress) == strings.ToLower(s.config.ProposerAddress) {
			if err := s.verifyPoSChallenge(c.BlockHeight); err != nil {
				return fmt.Errorf("Could not verify PoS challenge at block %d: %v", c.BlockHeight, err)
			}
		}

		// Fault any challenges which have not been verified in the deadline
		if blockHeight-c.BlockHeight >= int64(posVerificationDeadline) {
			if err = s.db.FaultPoSChallenge(ctx, c.BlockHeight); err != nil {
				return fmt.Errorf("Failed to update existing pos challenge at height %d: %v", c.BlockHeight, err)
			}
			// storage proofs are exempt because verifier faulted
			if err = s.db.ExemptStorageProofs(ctx, c.BlockHeight); err != nil {
				return fmt.Errorf("Failed to update existing storage proofs at height %d: %v", c.BlockHeight, err)
			}
		}
	}
	return nil
}

func (s *Server) sendPoSChallengeToMediorum(blockHash []byte, blockHeight int64, verifierAddr string, verifierPubKey string) {
	respChannel := make(chan pos.PoSResponse)
	posReq := pos.PoSRequest{
		Hash:     blockHash,
		Height:   blockHeight,
		Response: respChannel,
	}
	s.mediorumPoSChannel <- posReq

	timeout := time.After(mediorumPoSRequestTimeout)
	select {
	case response := <-respChannel:
		ctx := context.Background()

		// get validator nodes corresponding to mediorum's replica endpoints
		// TODO: check if mediorum normalizes these endpoints in a way that core does not
		nodes, err := s.db.GetNodesByEndpoints(ctx, response.Replicas)
		if err != nil {
			return fmt.Errorf("Failed to get all nodes for endpoints '%v': %v", replicaEndpoints, err)
		}
		prover_addresses := make([]string, 0, len(nodes))
		for _, n := range nodes {
			prover_addresses = append(prover_addresses, strings.ToLower(n.CometAddress))
		}

		// Add provers
		if err := s.db.UpdatePoSChallengeProvers(
			ctx,
			db.UpdatePoSChallengeProversParams{prover_addresses, blockHeight},
		); err != nil {
			s.logger.Error("Could not update existing PoS challenge", "hash", blockHash, "error", err)
		}

		// submit proof tx if we are part of the challenge
		if len(response.Proof) > 0 {
			err := s.submitStorageProofTx(blockHeight, blockHash, response.CID, prover_addresses, response.Proof, verifierAddr, verifierPubKey)
			if err != nil {
				s.logger.Error("Could not submit storage proof tx", "hash", blockHash, "error", err)
			}
		}

	case <-timeout:
		s.logger.Info("No response from mediorum for PoS challenge.")
	}
}

// TODO: delete the following once we establish it is unneeded
/*
	func (s *Server) updatePoSChallengeWithMediorumInfo(blockHeight int64, replicaEndpoints []string) error {
		dbTx, err := s.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("Could not initialize db transaction from pool: %v", err)
		}
		defer dbTx.Rollback(ctx)
		qtx := s.db.WithTx(dbTx)
		for _, n := range nodes {
			err := qtx.InsertStorageProof(
				ctx,
				db.InsertStorageProofParams{
					BlockHeight:    blockHeight,
					Address:        n.CometAddress,
					EncryptedProof: pgtype.Text{},
					DecryptedProof: pgtype.Text{},
				},
			)
			if err != nil {
				return fmt.Errorf("Could not insert empty storage proof for node %s", n.CometAddress)
			}
		}
		dbTx.Commit(ctx)
		return nil
	}
*/

func (s *Server) verifyPoSChallenge(blockHeight int64) error {
	ctx := context.Background()
	_, err := s.db.GetStorageProofs(ctx, blockHeight)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("Could not get storage proofs for block %d: %v", err)
	}
	return nil
}

type VerifierTuple struct {
	validator *db.CoreValidator
	score     []byte
}

type VerifierTuples []VerifierTuple

func (s VerifierTuples) Len() int      { return len(s) }
func (s VerifierTuples) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s VerifierTuples) Less(i, j int) bool {
	c := bytes.Compare(s[i].score, s[j].score)
	if c == 0 {
		return s[i].validator.CometAddress < s[j].validator.CometAddress
	}
	return c == -1
}

// Deterministically chooses a verifier for a Proof of Storage challenge based on the hash
func (s *Server) getPoSVerifierForChallenge(hash []byte) (db.CoreValidator, error) {
	var result db.CoreValidator
	hasher := sha256.New()
	validators, err := s.db.GetAllRegisteredNodesSorted(context.Background())
	if err != nil || len(validators) == 0 {
		return result, fmt.Errorf("Could not get registered nodes from db: %v", err)
	}
	tuples := make(VerifierTuples, len(validators))
	for i, validator := range validators {
		hasher.Reset()
		io.WriteString(hasher, validator.CometAddress)
		hasher.Write(hash)
		tuples[i] = VerifierTuple{&validator, hasher.Sum(nil)}
	}
	sort.Sort(tuples)
	result = *tuples[0].validator
	return result, nil
}

func generateRandomSecret() ([]byte, error) {
	secret := make([]byte, 16)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("failed to generate random secret: %v", err)
	}
	return secret, nil
}

func encryptStorageProof(secret, proof, blockHash []byte) ([]byte, error) {
	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %v", err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %v", err)
	}
	nonce := make([]byte, aesGCM.NonceSize())
	copy(nonce, blockHash)
	return aesGCM.Seal(nil, nonce[:aesGCM.NonceSize()], proof, nil), nil
}

func decryptStorageProof(encProof, secret, blockHash []byte) ([]byte, error) {
	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %v", err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %v", err)
	}
	nonce := make([]byte, aesGCM.NonceSize())
	copy(nonce, blockHash)
	proof, err := aesGCM.Open(nil, nonce, encProof, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %v", err)
	}
	return proof, nil
}

func encryptPoSSecret(pubKey, secret, blockHash []byte) ([]byte, []byte, error) {
	// 'Convert' Ed25519 public key to X25519 public key
	var xPubKey [32]byte
	copy(xPubKey[:], pubKey[:])

	// Generate ephemeral X25519 key pair
	var ephemeralPrivate, ephemeralPublic [32]byte
	if _, err := rand.Read(ephemeralPrivate[:]); err != nil {
		return nil, nil, fmt.Errorf("Could not generate ephemeral private key: %v", err)
	}
	curve25519.ScalarBaseMult(&ephemeralPublic, &ephemeralPrivate)

	// Compute symmetric key
	sharedSecret, err := curve25519.X25519(ephemeralPrivate[:], xPubKey[:])
	if err != nil {
		return nil, nil, fmt.Errorf("Could not compute shared secret: %v", err)
	}
	symmetricKey := sha256.Sum256(sharedSecret)

	// Encrypt proof secret with ChaCha20-Poly1305
	aead, err := chacha20poly1305.New(symmetricKey[:])
	if err != nil {
		return nil, nil, fmt.Errorf("Could not initialize cipher: %v", err)
	}
	nonce := make([]byte, aead.NonceSize())
	copy(nonce, blockHash)
	ciphertext := aead.Seal(nil, nonce, secret, nil)
	return ciphertext, ephemeralPublic[:], nil
}

func decryptPoSSecret(privateKey, secret, ephemeralPublicKey, blockHash []byte) ([]byte, error) {
	// 'Convert' ed25519 private key to X25519 private key
	var xKey [32]byte
	copy(xKey[:], privateKey[:])

	var ePubKey [32]byte
	copy(ePubKey[:], ephemeralPublicKey[:])

	sharedSecret, err := curve25519.X25519(xKey[:], ePubKey[:])
	if err != nil {
		return nil, fmt.Errorf("Could not compute shared secret: %v", err)
	}
	symmetricKey := sha256.Sum256(sharedSecret)

	aead, err := chacha20poly1305.New(symmetricKey[:])
	if err != nil {
		return nil, fmt.Errorf("Could not initialize cipher: %v", err)
	}
	nonce := make([]byte, aead.NonceSize())
	copy(nonce, blockHash)
	posSecret, err := aead.Open(nil, nonce, secret, nil)
	if err != nil {
		return nil, fmt.Errorf("Could not decrypt secret: %v", err)
	}
	return posSecret, nil
}
