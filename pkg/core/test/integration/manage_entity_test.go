package integration_test

import (
	"context"
	"time"

	"github.com/AudiusProject/audius-protocol/pkg/core/common"
	"github.com/AudiusProject/audius-protocol/pkg/core/gen/core_proto"
	"github.com/AudiusProject/audius-protocol/pkg/core/test/integration/utils"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/proto"
)

var _ = Describe("EntityManager", func() {
	It("sends and retrieves an entity manager transaction", func() {
		ctx := context.Background()

		sdk := utils.DiscoveryOne

		manageEntity := &core_proto.ManageEntityLegacy{
			UserId:     1,
			EntityType: "User",
			EntityId:   1,
			Action:     "Create",
			Metadata:   "some json",
			Signature:  "eip712",
			Nonce:      1,
			Signer:     "0x123",
		}

		signedManageEntity := &core_proto.SignedTransaction{
			RequestId: uuid.NewString(),
			Deadline:  1000000000000000000,
			Transaction: &core_proto.SignedTransaction_ManageEntity{
				ManageEntity: manageEntity,
			},
		}

		expectedTxHash, err := common.ToTxHash(signedManageEntity)
		Expect(err).To(BeNil())

		req := &core_proto.SendTransactionRequest{
			Transaction: signedManageEntity,
		}

		submitRes, err := sdk.SendTransaction(ctx, req)
		Expect(err).To(BeNil())

		txhash := submitRes.GetTxhash()
		Expect(expectedTxHash).To(Equal(txhash))

		time.Sleep(time.Second * 1)

		manageEntityRes, err := sdk.GetTransaction(ctx, &core_proto.GetTransactionRequest{Txhash: txhash})
		Expect(err).To(BeNil())

		Expect(proto.Equal(signedManageEntity, manageEntityRes.GetTransaction())).To(BeTrue())
	})
})
