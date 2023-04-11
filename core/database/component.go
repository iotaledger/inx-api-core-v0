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
	CoreComponent = &app.CoreComponent{
		Component: &app.Component{
			Name:     "database",
			DepsFunc: func(cDeps dependencies) { deps = cDeps },
			Params:   params,
			Provide:  provide,
			Run:      run,
		},
	}
}

type dependencies struct {
	dig.In
	Database        *database.Database
	Echo            *echo.Echo
	ShutdownHandler *shutdown.ShutdownHandler
}

var (
	CoreComponent *app.CoreComponent
	deps          dependencies
)

func provide(c *dig.Container) error {

	if err := c.Provide(func() (*database.Database, error) {
		CoreComponent.LogInfo("Setting up database ...")
		defer CoreComponent.LogInfo("Setting up database ... done!")

		tangleDatabase, err := engine.StoreWithDefaultSettings(ParamsDatabase.Tangle.Path, false, hivedb.EngineAuto, "tangle.db", engine.AllowedEnginesStorageAuto...)
		if err != nil {
			return nil, err
		}

		snapshotDatabase, err := engine.StoreWithDefaultSettings(ParamsDatabase.Snapshot.Path, false, hivedb.EngineAuto, "snapshot.db", engine.AllowedEnginesStorageAuto...)
		if err != nil {
			return nil, err
		}

		spentDatabase, err := engine.StoreWithDefaultSettings(ParamsDatabase.Spent.Path, false, hivedb.EngineAuto, "spent.db", engine.AllowedEnginesStorageAuto...)
		if err != nil {
			return nil, err
		}

		return database.New(tangleDatabase, snapshotDatabase, spentDatabase, ParamsDatabase.Debug)
	}); err != nil {
		return err
	}

	return nil
}

func run() error {

	if err := CoreComponent.Daemon().BackgroundWorker("Close database", func(ctx context.Context) {
		<-ctx.Done()

		CoreComponent.LogInfo("Syncing databases to disk ...")
		if err := deps.Database.CloseDatabases(); err != nil {
			CoreComponent.LogPanicf("Syncing databases to disk ... failed: %s", err)
		}
		CoreComponent.LogInfo("Syncing databases to disk ... done")
	}, daemon.PriorityStopDatabase); err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}

	return nil
}
