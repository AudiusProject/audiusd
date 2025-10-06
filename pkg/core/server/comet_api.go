// Request forward for the internal cometbft rpc. Debug info and to be turned off by default.
package server

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

var allowedRPCPrefixes = []string{
	"/status",
	"/health",
	"/block",
	"/block_results",
	"/validators",
	"/consensus_params",
	"/commit",
	"/tx",
	"/genesis",
	"/net_info",
}

func (s *Server) proxyCometRequest(c echo.Context) error {
	reqPath := strings.TrimPrefix(c.Request().RequestURI, "/core/crpc")

	// enforce allowlist
	allowed := false
	for _, p := range allowedRPCPrefixes {
		if strings.HasPrefix(reqPath, p) {
			allowed = true
			break
		}
	}
	if !allowed {
		s.logger.Warn("blocked unsafe comet rpc request", zap.String("path", reqPath))
		return respondWithError(c, http.StatusForbidden, "forbidden endpoint")
	}

	// only allow GET/HEAD
	if c.Request().Method != http.MethodGet && c.Request().Method != http.MethodHead {
		return respondWithError(c, http.StatusMethodNotAllowed, "method not allowed")
	}

	// build rpc target url
	rpcURL := strings.ReplaceAll(s.config.RPCladdr, "tcp", "http")
	targetURL := strings.TrimRight(rpcURL, "/") + reqPath

	s.logger.Info("proxy comet request",
		zap.String("target", targetURL),
		zap.String("method", c.Request().Method))

	// limit request size
	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, 1<<20) // 1MB

	req, err := http.NewRequestWithContext(c.Request().Context(), c.Request().Method, targetURL, c.Request().Body)
	if err != nil {
		s.logger.Error("failed to create comet request", zap.Error(err))
		return respondWithError(c, http.StatusInternalServerError, "internal error")
	}

	copyHeaders(c.Request().Header, req.Header)

	resp, err := httpClient.Do(req) // reuse your existing client
	if err != nil {
		s.logger.Error("failed to forward comet request", zap.Error(err))
		return respondWithError(c, http.StatusBadGateway, "comet node unavailable")
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		c.Response().Header()[k] = v
	}
	c.Response().WriteHeader(resp.StatusCode)

	if _, err := io.Copy(c.Response().Writer, resp.Body); err != nil {
		s.logger.Warn("failed to stream comet response", zap.Error(err))
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
