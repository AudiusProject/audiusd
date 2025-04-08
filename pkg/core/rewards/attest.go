package rewards

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

func (rs *RewardService) GetRewardById(rewardID string) (*Reward, error) {
	for _, reward := range rs.Rewards {
		if reward.RewardId == rewardID {
			return &reward, nil
		}
	}
	return nil, fmt.Errorf("reward %s not found", rewardID)
}

func (rs *RewardService) SignAttestation(attestationBytes []byte) (owner string, attestation string, err error) {
	/**
		def sign_attestation(attestation_bytes: bytes, private_key: str):
	    k = keys.PrivateKey(HexBytes(private_key))
	    to_sign_hash = Web3.keccak(attestation_bytes)
	    sig = k.sign_msg_hash(to_sign_hash)
	    return sig.to_hex()
	*/

	privateKey := rs.Config.EthereumKey
	pubKey := privateKey.Public()
	pubKeyECDSA, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return "", "", fmt.Errorf("failed to cast public key to ecdsa.PublicKey")
	}
	owner = crypto.PubkeyToAddress(*pubKeyECDSA).String()

	hash := crypto.Keccak256(attestationBytes)
	signature, err := crypto.Sign(hash, privateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign attestation: %w", err)
	}

	return owner, "0x" + hex.EncodeToString(signature), nil
}

func GetAttestationBytes(userWallet, rewardID, specifier, oracleAddress string, amount uint64) ([]byte, error) {
	/**
	  def _get_combined_id(self):
	      return f"{self.challenge_id}:{self.challenge_specifier}"
	*/
	combinedID := fmt.Sprintf("%s:%s", rewardID, specifier)

	/**
	WAUDIO_DECIMALS = 8

	def _get_encoded_amount(self):
		amt = int(self.amount) * 10**WAUDIO_DECIMALS
		return amt.to_bytes(8, byteorder="little")
	*/
	encodedAmount := amount * 1e8
	amountBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountBytes, uint64(encodedAmount))

	/**
	def get_attestation_bytes(self):
		user_bytes = to_bytes(hexstr=self.user_address)
		oracle_bytes = to_bytes(hexstr=self.oracle_address)
		combined_id_bytes = to_bytes(text=self._get_combined_id())
		amount_bytes = self._get_encoded_amount()
		items = [user_bytes, amount_bytes, combined_id_bytes, oracle_bytes]
		joined = to_bytes(text="_").join(items)
		return joined
	*/
	userBytes, err := hex.DecodeString(strings.TrimPrefix(userWallet, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode user wallet: %w", err)
	}
	oracleBytes, err := hex.DecodeString(strings.TrimPrefix(oracleAddress, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode oracle address: %w", err)
	}
	combinedIDBytes := []byte(combinedID)
	items := [][]byte{userBytes, amountBytes, combinedIDBytes, oracleBytes}
	attestationBytes := bytes.Join(items, []byte("_"))

	return attestationBytes, nil
}

func CompareClaimHash(userWallet, challengeId, challengeSpecifier, oracleAddress string, claimHash []byte) bool {
	claimDataHash := GetClaimDataHash(userWallet, challengeId, challengeSpecifier, oracleAddress)
	return bytes.Equal(claimDataHash, claimHash)
}

func RecoverWalletFromSignature(hash []byte, signature string) (string, error) {
	// Remove any existing 0x prefix
	sigHex := strings.TrimPrefix(signature, "0x")
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode signature: %w", err)
	}

	recoveredWallet, err := crypto.SigToPub(hash[:], sigBytes)
	if err != nil {
		return "", fmt.Errorf("failed to recover wallet from signature: %w", err)
	}

	return crypto.PubkeyToAddress(*recoveredWallet).String(), nil
}

func GetClaimDataHash(userWallet, challengeId, challengeSpecifier, oracleAddress string) []byte {
	claimData := fmt.Sprintf("%s_%s_%s_%s", userWallet, challengeId, challengeSpecifier, oracleAddress)
	return crypto.Keccak256([]byte(claimData))
}

func SignClaimDataHash(hash []byte, privateKey *ecdsa.PrivateKey) (string, error) {
	signature, err := crypto.Sign(hash, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign hash: %w", err)
	}
	return "0x" + hex.EncodeToString(signature), nil
}
