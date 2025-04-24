package middleware

import (
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/core/console/views"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func ErrorLoggerMiddleware(logger *common.Logger, views *views.Views) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			if err != nil {
				errorID := uuid.NewString()
				logger.Error("error occured", "id", errorID, "error", err, "path", c.Path())
				return views.RenderErrorView(c, errorID)
			}
			return nil
		}
	}
}
