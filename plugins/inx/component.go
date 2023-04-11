package inx

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/dig"

	"github.com/iotaledger/hive.go/app"
	"github.com/iotaledger/hive.go/app/shutdown"
	"github.com/iotaledger/inx-api-core-v0/pkg/daemon"
	"github.com/iotaledger/inx-api-core-v0/pkg/server"
	"github.com/iotaledger/inx-app/pkg/nodebridge"
)

const (
	APIRoute = "core/v0"
)

func init() {
	Plugin = &app.Plugin{
		Component: &app.Component{
			Name:      "INX",
			DepsFunc:  func(cDeps dependencies) { deps = cDeps },
			Params:    params,
			Provide:   provide,
			Configure: configure,
			Run:       run,
		},
		IsEnabled: func() bool {
			return ParamsINX.Enabled
		},
	}
}

type dependencies struct {
	dig.In
	NodeBridge              *nodebridge.NodeBridge
	ShutdownHandler         *shutdown.ShutdownHandler
	RestAPIBindAddress      string `name:"restAPIBindAddress"`
	RestAPIAdvertiseAddress string `name:"restAPIAdvertiseAddress"`
}

var (
	Plugin *app.Plugin
	deps   dependencies
)

func provide(c *dig.Container) error {
	return c.Provide(func() *nodebridge.NodeBridge {
		return nodebridge.NewNodeBridge(Plugin.Logger(), nodebridge.WithTargetNetworkName(ParamsINX.TargetNetworkName))
	})
}

func configure() error {
	if err := deps.NodeBridge.Connect(
		Plugin.Daemon().ContextStopped(),
		ParamsINX.Address,
		ParamsINX.MaxConnectionAttempts,
	); err != nil {
		Plugin.LogErrorfAndExit("failed to connect via INX: %s", err.Error())
	}

	return nil
}

func run() error {
	if err := Plugin.Daemon().BackgroundWorker("INX", func(ctx context.Context) {
		Plugin.LogInfo("Starting NodeBridge ...")
		deps.NodeBridge.Run(ctx)
		Plugin.LogInfo("Stopped NodeBridge")

		if !errors.Is(ctx.Err(), context.Canceled) {
			deps.ShutdownHandler.SelfShutdown("INX connection to node dropped", true)
		}
	}, daemon.PriorityDisconnectINX); err != nil {
		Plugin.LogPanicf("failed to start worker: %s", err)
	}

	if err := Plugin.Daemon().BackgroundWorker("INX-RestAPI", func(ctx context.Context) {
		ctxRegister, cancelRegister := context.WithTimeout(ctx, 5*time.Second)

		advertisedAddress := deps.RestAPIBindAddress
		if deps.RestAPIAdvertiseAddress != "" {
			advertisedAddress = deps.RestAPIAdvertiseAddress
		}

		if err := deps.NodeBridge.RegisterAPIRoute(ctxRegister, APIRoute, advertisedAddress); err != nil {
			Plugin.LogErrorfAndExit("Registering INX api route failed: %s", err)
		}
		cancelRegister()

		<-ctx.Done()

		ctxUnregister, cancelUnregister := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelUnregister()

		//nolint:contextcheck // false positive
		if err := deps.NodeBridge.UnregisterAPIRoute(ctxUnregister, APIRoute); err != nil {
			Plugin.LogWarnf("Unregistering INX api route failed: %s", err)
		}
	}, daemon.PriorityStopDatabaseAPIINX); err != nil {
		Plugin.LogPanicf("failed to start worker: %s", err)
	}

	return nil
}
