// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: etl/v1/service.proto

package v1connect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	v1 "github.com/AudiusProject/audiusd/pkg/api/etl/v1"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect.IsAtLeastVersion1_13_0

const (
	// ETLServiceName is the fully-qualified name of the ETLService service.
	ETLServiceName = "etl.v1.ETLService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// ETLServicePingProcedure is the fully-qualified name of the ETLService's Ping RPC.
	ETLServicePingProcedure = "/etl.v1.ETLService/Ping"
	// ETLServiceGetHealthProcedure is the fully-qualified name of the ETLService's GetHealth RPC.
	ETLServiceGetHealthProcedure = "/etl.v1.ETLService/GetHealth"
	// ETLServiceGetPlaysProcedure is the fully-qualified name of the ETLService's GetPlays RPC.
	ETLServiceGetPlaysProcedure = "/etl.v1.ETLService/GetPlays"
)

// ETLServiceClient is a client for the etl.v1.ETLService service.
type ETLServiceClient interface {
	Ping(context.Context, *connect.Request[v1.PingRequest]) (*connect.Response[v1.PingResponse], error)
	GetHealth(context.Context, *connect.Request[v1.GetHealthRequest]) (*connect.Response[v1.GetHealthResponse], error)
	GetPlays(context.Context, *connect.Request[v1.GetPlaysRequest]) (*connect.Response[v1.GetPlaysResponse], error)
}

// NewETLServiceClient constructs a client for the etl.v1.ETLService service. By default, it uses
// the Connect protocol with the binary Protobuf Codec, asks for gzipped responses, and sends
// uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or
// connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewETLServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) ETLServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	eTLServiceMethods := v1.File_etl_v1_service_proto.Services().ByName("ETLService").Methods()
	return &eTLServiceClient{
		ping: connect.NewClient[v1.PingRequest, v1.PingResponse](
			httpClient,
			baseURL+ETLServicePingProcedure,
			connect.WithSchema(eTLServiceMethods.ByName("Ping")),
			connect.WithClientOptions(opts...),
		),
		getHealth: connect.NewClient[v1.GetHealthRequest, v1.GetHealthResponse](
			httpClient,
			baseURL+ETLServiceGetHealthProcedure,
			connect.WithSchema(eTLServiceMethods.ByName("GetHealth")),
			connect.WithClientOptions(opts...),
		),
		getPlays: connect.NewClient[v1.GetPlaysRequest, v1.GetPlaysResponse](
			httpClient,
			baseURL+ETLServiceGetPlaysProcedure,
			connect.WithSchema(eTLServiceMethods.ByName("GetPlays")),
			connect.WithClientOptions(opts...),
		),
	}
}

// eTLServiceClient implements ETLServiceClient.
type eTLServiceClient struct {
	ping      *connect.Client[v1.PingRequest, v1.PingResponse]
	getHealth *connect.Client[v1.GetHealthRequest, v1.GetHealthResponse]
	getPlays  *connect.Client[v1.GetPlaysRequest, v1.GetPlaysResponse]
}

// Ping calls etl.v1.ETLService.Ping.
func (c *eTLServiceClient) Ping(ctx context.Context, req *connect.Request[v1.PingRequest]) (*connect.Response[v1.PingResponse], error) {
	return c.ping.CallUnary(ctx, req)
}

// GetHealth calls etl.v1.ETLService.GetHealth.
func (c *eTLServiceClient) GetHealth(ctx context.Context, req *connect.Request[v1.GetHealthRequest]) (*connect.Response[v1.GetHealthResponse], error) {
	return c.getHealth.CallUnary(ctx, req)
}

// GetPlays calls etl.v1.ETLService.GetPlays.
func (c *eTLServiceClient) GetPlays(ctx context.Context, req *connect.Request[v1.GetPlaysRequest]) (*connect.Response[v1.GetPlaysResponse], error) {
	return c.getPlays.CallUnary(ctx, req)
}

// ETLServiceHandler is an implementation of the etl.v1.ETLService service.
type ETLServiceHandler interface {
	Ping(context.Context, *connect.Request[v1.PingRequest]) (*connect.Response[v1.PingResponse], error)
	GetHealth(context.Context, *connect.Request[v1.GetHealthRequest]) (*connect.Response[v1.GetHealthResponse], error)
	GetPlays(context.Context, *connect.Request[v1.GetPlaysRequest]) (*connect.Response[v1.GetPlaysResponse], error)
}

// NewETLServiceHandler builds an HTTP handler from the service implementation. It returns the path
// on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewETLServiceHandler(svc ETLServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	eTLServiceMethods := v1.File_etl_v1_service_proto.Services().ByName("ETLService").Methods()
	eTLServicePingHandler := connect.NewUnaryHandler(
		ETLServicePingProcedure,
		svc.Ping,
		connect.WithSchema(eTLServiceMethods.ByName("Ping")),
		connect.WithHandlerOptions(opts...),
	)
	eTLServiceGetHealthHandler := connect.NewUnaryHandler(
		ETLServiceGetHealthProcedure,
		svc.GetHealth,
		connect.WithSchema(eTLServiceMethods.ByName("GetHealth")),
		connect.WithHandlerOptions(opts...),
	)
	eTLServiceGetPlaysHandler := connect.NewUnaryHandler(
		ETLServiceGetPlaysProcedure,
		svc.GetPlays,
		connect.WithSchema(eTLServiceMethods.ByName("GetPlays")),
		connect.WithHandlerOptions(opts...),
	)
	return "/etl.v1.ETLService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case ETLServicePingProcedure:
			eTLServicePingHandler.ServeHTTP(w, r)
		case ETLServiceGetHealthProcedure:
			eTLServiceGetHealthHandler.ServeHTTP(w, r)
		case ETLServiceGetPlaysProcedure:
			eTLServiceGetPlaysHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedETLServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedETLServiceHandler struct{}

func (UnimplementedETLServiceHandler) Ping(context.Context, *connect.Request[v1.PingRequest]) (*connect.Response[v1.PingResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("etl.v1.ETLService.Ping is not implemented"))
}

func (UnimplementedETLServiceHandler) GetHealth(context.Context, *connect.Request[v1.GetHealthRequest]) (*connect.Response[v1.GetHealthResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("etl.v1.ETLService.GetHealth is not implemented"))
}

func (UnimplementedETLServiceHandler) GetPlays(context.Context, *connect.Request[v1.GetPlaysRequest]) (*connect.Response[v1.GetPlaysResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("etl.v1.ETLService.GetPlays is not implemented"))
}
