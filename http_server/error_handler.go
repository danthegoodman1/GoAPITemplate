package http_server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

func customHTTPErrorHandler(err error, c echo.Context) {
	var he *echo.HTTPError
	if errors.As(err, &he) {
		var message string
		if he.Message == nil {
			message = http.StatusText(he.Code)
		} else if msg, ok := he.Message.(string); ok {
			message = msg
		} else {
			message = fmt.Sprint(he.Message)
		}
		c.String(he.Code, message)
		return
	}

	logger := zerolog.Ctx(c.Request().Context())
	logger.Error().Err(err).Msg("unhandled internal error")

	c.String(http.StatusInternalServerError, "Something went wrong internally, an error has been logged")
}
