// Request forward for the internal cometbft rpc. Debug info and to be turned off by default.
package server

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/AudiusProject/audiusd/pkg/core/config"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func (s *Server) proxyCometRequest(c echo.Context) error {
	if !s.config.StateSync.ServeSnapshots {
		return errors.New("")
	}

	// Create HTTP client with Unix socket transport
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				dialer := net.Dialer{}
				return dialer.DialContext(ctx, "unix", config.CometRPCSocket)
			},
		},
	}

	s.logger.Info("request", zap.String("socket", config.CometRPCSocket), zap.String("method", c.Request().Method), zap.String("url", c.Request().RequestURI))

	// For Unix sockets, the host is ignored, but we need to provide one
	path := "http://localhost" + strings.TrimPrefix(c.Request().RequestURI, "/core/crpc")

	req, err := http.NewRequest(c.Request().Method, path, c.Request().Body)
	if err != nil {
		s.logger.Error("failed to create internal comet api request", zap.Error(err))
		return respondWithError(c, http.StatusInternalServerError, "failed to create internal comet request")
	}

	copyHeaders(c.Request().Header, req.Header)

	resp, err := httpClient.Do(req)
	if err != nil {
		s.logger.Error("failed to forward comet api request", zap.Error(err))
		return respondWithError(c, http.StatusInternalServerError, "failed to forward request")
	}
	defer resp.Body.Close()

	c.Response().Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	c.Response().WriteHeader(resp.StatusCode)
	_, err = io.Copy(c.Response().Writer, resp.Body)
	if err != nil {
		return respondWithError(c, http.StatusInternalServerError, "failed to stream response")
	}

	return nil
}

func copyHeaders(source http.Header, destination http.Header) {
	for k, v := range source {
		destination[k] = v
	}
}

func respondWithError(c echo.Context, statusCode int, message string) error {
	return c.JSON(statusCode, map[string]string{"error": message})
}
