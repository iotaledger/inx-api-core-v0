package inx

import (
	"context"
	"time"

	"go.uber.org/dig"

	"github.com/iotaledger/hive.go/app"
	"github.com/iotaledger/hive.go/app/shutdown"
	"github.com/iotaledger/hive.go/ierrors"
	"github.com/iotaledger/inx-api-core-v0/pkg/daemon"
	"github.com/iotaledger/inx-api-core-v0/pkg/server"
	"github.com/iotaledger/inx-app/pkg/nodebridge"
)

const (
	APIRoute = "core/v0"
)

func init() {
	Component = &app.Component{
		Name:      "INX",
		DepsFunc:  func(cDeps dependencies) { deps = cDeps },
		Params:    params,
		IsEnabled: func(_ *dig.Container) bool { return ParamsINX.Enabled },
		Provide:   provide,
		Configure: configure,
		Run:       run,
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
	Component *app.Component
	deps      dependencies
)

func provide(c *dig.Container) error {
	return c.Provide(func() *nodebridge.NodeBridge {
		return nodebridge.NewNodeBridge(Component.Logger(), nodebridge.WithTargetNetworkName(ParamsINX.TargetNetworkName))
	})
}

func configure() error {
	if err := deps.NodeBridge.Connect(
		Component.Daemon().ContextStopped(),
		ParamsINX.Address,
		ParamsINX.MaxConnectionAttempts,
	); err != nil {
		Component.LogErrorfAndExit("failed to connect via INX: %s", err.Error())
	}

	return nil
}

func run() error {
	if err := Component.Daemon().BackgroundWorker("INX", func(ctx context.Context) {
		Component.LogInfo("Starting NodeBridge ...")
		deps.NodeBridge.Run(ctx)
		Component.LogInfo("Stopped NodeBridge")

		if !ierrors.Is(ctx.Err(), context.Canceled) {
			deps.ShutdownHandler.SelfShutdown("INX connection to node dropped", true)
		}
	}, daemon.PriorityDisconnectINX); err != nil {
		Component.LogPanicf("failed to start worker: %s", err)
	}

	if err := Component.Daemon().BackgroundWorker("INX-RestAPI", func(ctx context.Context) {
		ctxRegister, cancelRegister := context.WithTimeout(ctx, 5*time.Second)

		advertisedAddress := deps.RestAPIBindAddress
		if deps.RestAPIAdvertiseAddress != "" {
			advertisedAddress = deps.RestAPIAdvertiseAddress
		}

		if err := deps.NodeBridge.RegisterAPIRoute(ctxRegister, APIRoute, advertisedAddress, server.APIRoute); err != nil {
			Component.LogErrorfAndExit("Registering INX api route failed: %s", err)
		}
		cancelRegister()

		<-ctx.Done()

		ctxUnregister, cancelUnregister := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelUnregister()

		//nolint:contextcheck // false positive
		if err := deps.NodeBridge.UnregisterAPIRoute(ctxUnregister, APIRoute); err != nil {
			Component.LogWarnf("Unregistering INX api route failed: %s", err)
		}
	}, daemon.PriorityStopDatabaseAPIINX); err != nil {
		Component.LogPanicf("failed to start worker: %s", err)
	}

	return nil
}
