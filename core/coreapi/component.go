package coreapi

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/dig"

	"github.com/iotaledger/hive.go/app"
	"github.com/iotaledger/inx-api-core-v0/pkg/daemon"
	"github.com/iotaledger/inx-api-core-v0/pkg/database"
	"github.com/iotaledger/inx-api-core-v0/pkg/server"
	"github.com/iotaledger/inx-app/pkg/httpserver"
)

func init() {
	CoreComponent = &app.CoreComponent{
		Component: &app.Component{
			Name:           "CoreAPIV0",
			DepsFunc:       func(cDeps dependencies) { deps = cDeps },
			Params:         params,
			InitConfigPars: initConfigPars,
			Provide:        provide,
			Run:            run,
		},
	}
}

type dependencies struct {
	dig.In
	AppInfo  *app.Info
	Database *database.Database
	Echo     *echo.Echo
}

var (
	CoreComponent *app.CoreComponent
	deps          dependencies
)

func initConfigPars(c *dig.Container) error {

	type cfgResult struct {
		dig.Out
		RestAPIBindAddress      string `name:"restAPIBindAddress"`
		RestAPIAdvertiseAddress string `name:"restAPIAdvertiseAddress"`
	}

	if err := c.Provide(func() cfgResult {
		return cfgResult{
			RestAPIBindAddress:      ParamsRestAPI.BindAddress,
			RestAPIAdvertiseAddress: ParamsRestAPI.AdvertiseAddress,
		}
	}); err != nil {
		CoreComponent.LogPanic(err)
	}

	return nil
}

func provide(c *dig.Container) error {

	if err := c.Provide(func() *echo.Echo {
		e := httpserver.NewEcho(
			CoreComponent.Logger(),
			nil,
			ParamsRestAPI.DebugRequestLoggerEnabled,
		)
		e.Use(middleware.Gzip())
		e.Use(middleware.BodyLimit(ParamsRestAPI.Limits.MaxBodyLength))

		return e
	}); err != nil {
		return err
	}

	return nil
}

func run() error {

	// create a background worker that handles the API
	if err := CoreComponent.Daemon().BackgroundWorker("API", func(ctx context.Context) {
		CoreComponent.LogInfo("Starting API server ...")

		swagger := server.CreateEchoSwagger(deps.Echo, deps.AppInfo.Version, ParamsRestAPI.SwaggerEnabled)

		//nolint:contextcheck //false positive
		_ = server.NewDatabaseServer(
			swagger,
			deps.AppInfo,
			deps.Database,
			ParamsRestAPI.Limits.MaxResults,
		)

		deps.Echo.Server.BaseContext = func(l net.Listener) context.Context {
			// set BaseContext to be the same as the worker,
			// so that requests being processed don't hang the shutdown procedure
			return ctx
		}

		go func() {
			CoreComponent.LogInfof("You can now access the API using: http://%s", ParamsRestAPI.BindAddress)
			if err := deps.Echo.Start(ParamsRestAPI.BindAddress); err != nil && !errors.Is(err, http.ErrServerClosed) {
				CoreComponent.LogErrorfAndExit("Stopped REST-API server due to an error (%s)", err)
			}
		}()

		CoreComponent.LogInfo("Starting API server ... done")
		<-ctx.Done()
		CoreComponent.LogInfo("Stopping API server ...")

		shutdownCtx, shutdownCtxCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCtxCancel()

		//nolint:contextcheck // false positive
		if err := deps.Echo.Shutdown(shutdownCtx); err != nil {
			CoreComponent.LogWarn(err)
		}

		CoreComponent.LogInfo("Stopping API server... done")
	}, daemon.PriorityStopDatabaseAPI); err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}

	return nil
}
