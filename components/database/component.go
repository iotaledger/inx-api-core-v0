package database

import (
	"context"

	"github.com/labstack/echo/v4"
	"go.uber.org/dig"

	"github.com/iotaledger/hive.go/app"
	"github.com/iotaledger/hive.go/app/shutdown"
	hivedb "github.com/iotaledger/hive.go/kvstore/database"
	"github.com/iotaledger/inx-api-core-v0/pkg/daemon"
	"github.com/iotaledger/inx-api-core-v0/pkg/database"
	"github.com/iotaledger/inx-api-core-v0/pkg/database/engine"
)

const (
	DBVersion uint32 = 2
)

func init() {
	Component = &app.Component{
		Name:     "database",
		DepsFunc: func(cDeps dependencies) { deps = cDeps },
		Params:   params,
		Provide:  provide,
		Run:      run,
	}
}

type dependencies struct {
	dig.In
	Database        *database.Database
	Echo            *echo.Echo
	ShutdownHandler *shutdown.ShutdownHandler
}

var (
	Component *app.Component
	deps      dependencies
)

func provide(c *dig.Container) error {
	return c.Provide(func() (*database.Database, error) {
		Component.LogInfo("Setting up database ...")
		defer Component.LogInfo("Setting up database ... done!")

		tangleDatabase, err := engine.StoreWithDefaultSettings(ParamsDatabase.Tangle.Path, false, hivedb.EngineAuto, engine.AllowedEnginesStorageAuto...)
		if err != nil {
			return nil, err
		}

		snapshotDatabase, err := engine.StoreWithDefaultSettings(ParamsDatabase.Snapshot.Path, false, hivedb.EngineAuto, engine.AllowedEnginesStorageAuto...)
		if err != nil {
			return nil, err
		}

		spentDatabase, err := engine.StoreWithDefaultSettings(ParamsDatabase.Spent.Path, false, hivedb.EngineAuto, engine.AllowedEnginesStorageAuto...)
		if err != nil {
			return nil, err
		}

		return database.New(tangleDatabase, snapshotDatabase, spentDatabase, ParamsDatabase.Debug)
	})
}

func run() error {

	if err := Component.Daemon().BackgroundWorker("Close database", func(ctx context.Context) {
		<-ctx.Done()

		Component.LogInfo("Syncing databases to disk ...")
		if err := deps.Database.CloseDatabases(); err != nil {
			Component.LogPanicf("Syncing databases to disk ... failed: %s", err)
		}
		Component.LogInfo("Syncing databases to disk ... done")
	}, daemon.PriorityStopDatabase); err != nil {
		Component.LogPanicf("failed to start worker: %s", err)
	}

	return nil
}
