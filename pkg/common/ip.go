package common

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

const ClientIPKey = "client-ip"

func GetClientIP(ctx context.Context) string {
	val := ctx.Value(ClientIPKey)
	if val == nil {
		return ""
	}

	ip, ok := val.(string)
	if !ok {
		return ""
	}

	return ip
}

func InjectRealIP() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := ExtractRealIP(c.Request())
			ctx := context.WithValue(c.Request().Context(), ClientIPKey, ip)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

func ExtractRealIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Only use the first IP in the list
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
