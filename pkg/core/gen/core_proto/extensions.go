// methods on grpc methods that help with functionality
package core_proto

import (
	"github.com/cometbft/cometbft/libs/bytes"
	"github.com/cometbft/cometbft/types"
	"google.golang.org/protobuf/proto"
)

// impl copied from common.ToTxHash()
func (tx *SignedTransaction) GetTxHash() string {
	b, err := proto.Marshal(tx)
	if err != nil {
		return ""
	}

	transaction := types.Tx(b)
	hash := transaction.Hash()
	hexBytes := bytes.HexBytes(hash)
	hashStr := hexBytes.String()

	return hashStr
}
