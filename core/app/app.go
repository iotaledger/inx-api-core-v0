package app

import (
	"github.com/iotaledger/hive.go/app"
	"github.com/iotaledger/hive.go/app/components/profiling"
	"github.com/iotaledger/hive.go/app/components/shutdown"
	"github.com/iotaledger/inx-api-core-v0/core/coreapi"
	"github.com/iotaledger/inx-api-core-v0/core/database"
	"github.com/iotaledger/inx-api-core-v0/plugins/inx"
	"github.com/iotaledger/inx-api-core-v0/plugins/prometheus"
)

var (
	// Name of the app.
	Name = "inx-api-core-v0"

	// Version of the app.
	Version = "1.0.0-rc.2"
)

func App() *app.App {
	return app.New(Name, Version,
		app.WithInitComponent(InitComponent),
		app.WithCoreComponents([]*app.CoreComponent{
			shutdown.CoreComponent,
			database.CoreComponent,
			coreapi.CoreComponent,
		}...),
		app.WithPlugins([]*app.Plugin{
			inx.Plugin,
			profiling.Plugin,
			prometheus.Plugin,
		}...),
	)
}

var (
	InitComponent *app.InitComponent
)

func init() {
	InitComponent = &app.InitComponent{
		Component: &app.Component{
			Name: "App",
		},
		NonHiddenFlags: []string{
			"config",
			"help",
			"version",
		},
	}
}
