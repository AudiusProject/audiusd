package service

import (
	"github.com/AudiusProject/audiusd/pkg/core/common"
	"github.com/labstack/echo/v4"
)

type Service[T any] interface {
	// method that creates the service with defaults
	// registers routes to the echo instance, etc
	Initialize(e *echo.Echo, l *common.Logger) (T, error)
	// starts the service
	Start() error
	// gracefully stops the service
	Stop() error
}
