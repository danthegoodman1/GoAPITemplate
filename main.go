package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/danthegoodman1/GoAPITemplate/observability"
	"github.com/danthegoodman1/GoAPITemplate/temporal"
	"github.com/joho/godotenv"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danthegoodman1/GoAPITemplate/gologger"
	"github.com/danthegoodman1/GoAPITemplate/http_server"
	"github.com/danthegoodman1/GoAPITemplate/migrations"
	"github.com/danthegoodman1/GoAPITemplate/pg"
	"github.com/danthegoodman1/GoAPITemplate/utils"
)

var logger = gologger.NewLogger()

func main() {
	if _, err := os.Stat(".env"); err == nil {
		err = godotenv.Load()
		if err != nil {
			logger.Fatal().Err(err).Msg("error loading .env file, exiting")
			return
		}
	}
	logger.Debug().Msg("starting unnamed api")

	if err := pg.ConnectToDB(); err != nil {
		logger.Fatal().Err(err).Msg("error connecting to PG")
		return
	}

	err := migrations.CheckMigrations(utils.PG_DSN)
	if err != nil {
		logger.Fatal().Err(err).Msg("Error checking migrations")
		return
	}

	prometheusReporter := observability.NewPrometheusReporter()
	go func() {
		err := observability.StartInternalHTTPServer(":8042", prometheusReporter)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("internal server couldn't start")
			return
		}
	}()

	err = temporal.Run(context.Background(), prometheusReporter)
	if err != nil {
		logger.Fatal().Err(err).Msg("Temporal init error")
		return
	}

	httpServer := http_server.StartHTTPServer()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	logger.Warn().Msg("received shutdown signal!")

	// For AWS ALB needing some time to de-register pod
	// Convert the time to seconds
	sleepTime := utils.GetEnvOrDefaultInt("SHUTDOWN_SLEEP_SEC", 0)
	logger.Info().Msg(fmt.Sprintf("sleeping for %ds before exiting", sleepTime))

	time.Sleep(time.Second * time.Duration(sleepTime))
	logger.Info().Msg(fmt.Sprintf("slept for %ds, exiting", sleepTime))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("failed to shutdown HTTP server")
	} else {
		logger.Info().Msg("successfully shutdown HTTP server")
	}
}
