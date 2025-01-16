package sdk

import (
	"context"
	"time"

	"github.com/AudiusProject/audiusd/pkg/core/gen/core_openapi"
	"github.com/AudiusProject/audiusd/pkg/core/gen/core_openapi/protocol"
	"github.com/AudiusProject/audiusd/pkg/core/gen/core_proto"
	"github.com/cometbft/cometbft/rpc/client/http"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Sdk struct {
	logger   Logger
	useHttps bool
	privKey  string

	retries int
	delay   time.Duration

	OAPIEndpoint string
	GRPCEndpoint string
	JRPCEndpoint string
	protocol.ClientService
	core_proto.ProtocolClient
	http.HTTP
}

func defaultSdk() *Sdk {
	return &Sdk{
		logger:  NewNoOpLogger(),
		retries: 10,
		delay:   3 * time.Second,
	}
}

func initSdk(sdk *Sdk) error {
	ctx := context.Background()
	// TODO: add default environment here if not set

	// TODO: add node selection logic here, based on environment, if endpoint not configured

	g, ctx := errgroup.WithContext(ctx)

	if sdk.OAPIEndpoint != "" {
		g.Go(func() error {
			transport := core_openapi.DefaultTransportConfig().WithHost(sdk.OAPIEndpoint)
			if sdk.useHttps {
				transport.WithSchemes([]string{"https"})
			}

			retries := sdk.retries

			client := core_openapi.NewHTTPClientWithConfig(nil, transport)
			for tries := retries; tries >= 0; tries-- {
				_, err := client.Protocol.ProtocolPing(protocol.NewProtocolPingParams())
				if err == nil {
					break
				}

				if tries == 0 {
					sdk.logger.Error("exhausted openapi retries", "error", err, "endpoint", sdk.OAPIEndpoint)
					return err
				}

				time.Sleep(sdk.delay)
			}

			sdk.ClientService = client.Protocol
			return nil
		})
	}

	// initialize grpc client
	if sdk.GRPCEndpoint != "" {
		g.Go(func() error {
			grpcConn, err := grpc.NewClient(sdk.GRPCEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return err
			}

			grpcClient := core_proto.NewProtocolClient(grpcConn)

			for tries := sdk.retries; tries >= 0; tries-- {
				_, err := grpcClient.Ping(ctx, &core_proto.PingRequest{})
				if err == nil {
					break
				}

				if tries == 0 {
					sdk.logger.Error("exhausted grpc retries", "error", err, "endpoint", sdk.GRPCEndpoint)
					return err
				}

				time.Sleep(sdk.delay)
			}

			sdk.ProtocolClient = grpcClient
			return nil
		})
	}

	// initialize jsonrpc client
	if sdk.JRPCEndpoint != "" {
		g.Go(func() error {
			jrpcConn, err := http.New(sdk.JRPCEndpoint)
			if err != nil {
				return err
			}

			for tries := sdk.retries; tries >= 0; tries-- {
				_, err := jrpcConn.Health(ctx)
				if err == nil {
					break
				}

				if tries == 0 {
					sdk.logger.Error("exhausted jrpc retries", "error", err, "endpoint", sdk.GRPCEndpoint)
					return err
				}

				time.Sleep(sdk.delay)
			}

			sdk.HTTP = *jrpcConn
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		sdk.logger.Error("init sdk error", "error", err)
		return err
	}

	if sdk.privKey == "" {
		sdk.logger.Info("private key not supplied to sdk, only reads allowed")
	}

	return nil
}
