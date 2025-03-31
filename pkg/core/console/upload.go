package console

import "github.com/labstack/echo/v4"

func (cs *Console) uploadPage(c echo.Context) error {
	return cs.views.RenderUploadPageView(c)
}
