package http_server

import (
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"net/http"
)

func customHTTPErrorHandler(err error, c echo.Context) {
	if he, ok := err.(*echo.HTTPError); ok {
		c.String(he.Code, he.Message.(string))
		return
	}

	logger := zerolog.Ctx(c.Request().Context())
	logger.Error().Err(err).Msg("unhandled internal error")

	c.String(http.StatusInternalServerError, "Something went wrong internally, an error has been logged")
}
