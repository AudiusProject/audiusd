package server

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	corev1 "github.com/AudiusProject/audiusd/pkg/gen/core/v1"
	"github.com/AudiusProject/audiusd/pkg/gen/core/v1/corev1connect"
	"github.com/labstack/echo/v4"
)

type Server struct {
	// ...
}

func NewServer() *Server {
	return &Server{}
}

type CoreServer struct {
}

// ForwardTransaction implements corev1connect.CoreServiceHandler.
func (c *CoreServer) ForwardTransaction(context.Context, *connect.Request[corev1.ForwardTransactionRequest]) (*connect.Response[corev1.ForwardTransactionResponse], error) {
	panic("unimplemented")
}

// GetBlock implements corev1connect.CoreServiceHandler.
func (c *CoreServer) GetBlock(context.Context, *connect.Request[corev1.GetBlockRequest]) (*connect.Response[corev1.GetBlockResponse], error) {
	panic("unimplemented")
}

// GetTransaction implements corev1connect.CoreServiceHandler.
func (c *CoreServer) GetTransaction(context.Context, *connect.Request[corev1.GetTransactionRequest]) (*connect.Response[corev1.GetTransactionResponse], error) {
	panic("unimplemented")
}

// HealthCheck implements corev1connect.CoreServiceHandler.
func (c *CoreServer) HealthCheck(context.Context, *connect.Request[corev1.HealthCheckRequest]) (*connect.Response[corev1.HealthCheckResponse], error) {
	panic("unimplemented")
}

// Ping implements corev1connect.CoreServiceHandler.
func (c *CoreServer) Ping(context.Context, *connect.Request[corev1.PingRequest]) (*connect.Response[corev1.PingResponse], error) {
	panic("unimplemented")
}

// SendTransaction implements corev1connect.CoreServiceHandler.
func (c *CoreServer) SendTransaction(context.Context, *connect.Request[corev1.SendTransactionRequest]) (*connect.Response[corev1.SendTransactionResponse], error) {
	panic("unimplemented")
}

var _ corev1connect.CoreServiceHandler = (*CoreServer)(nil)

func (s *Server) Start() error {
	e := echo.New()

	cs := &CoreServer{}
	mux := http.NewServeMux()
	path, handler := corev1connect.NewCoreServiceHandler(cs)

	mux.Handle(path, handler)
	e.Any("/*", echo.WrapHandler(mux))

	return e.Start(":8080")
}
