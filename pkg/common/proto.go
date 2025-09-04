package common

import (
	"github.com/cometbft/cometbft/libs/bytes"
	"github.com/cometbft/cometbft/types"
	"google.golang.org/protobuf/proto"
)

type TxHash = string

func ToTxHash(msg proto.Message) (TxHash, error) {
	b, err := proto.Marshal(msg)
	if err != nil {
		return "", err
	}

	tx := types.Tx(b)
	hash := tx.Hash()
	hexBytes := bytes.HexBytes(hash)
	hashStr := hexBytes.String()

	return hashStr, nil
}

// ToTxHashFromBytes creates a transaction hash from raw transaction bytes
// using CometBFT's hashing utilities for consistency with block sync
func ToTxHashFromBytes(txBytes []byte) TxHash {
	tx := types.Tx(txBytes)
	hash := tx.Hash()
	return bytes.HexBytes(hash).String()
}
