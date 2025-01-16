package integration_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/AudiusProject/audiusd/pkg/core/gen/core_proto"
	"github.com/AudiusProject/audiusd/pkg/core/test/integration/utils"
)

var _ = Describe("Sdk", func() {
	It("connects to both rpc endpoints", func() {
		ctx := context.Background()

		sdk := utils.ContentOne

		// test jsonrpc health route
		_, err := sdk.Health(ctx)
		Expect(err).To(BeNil())

		// test grpc hello route
		res, err := sdk.Ping(ctx, &core_proto.PingRequest{})
		Expect(err).To(BeNil())
		Expect(res.GetMessage()).To(Equal("pong"))
	})
})
