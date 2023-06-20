package app

import (
	"github.com/iotaledger/hive.go/app"
	"github.com/iotaledger/hive.go/app/components/profiling"
	"github.com/iotaledger/hive.go/app/components/shutdown"
	"github.com/iotaledger/inx-api-core-v0/components/coreapi"
	"github.com/iotaledger/inx-api-core-v0/components/database"
	"github.com/iotaledger/inx-api-core-v0/components/inx"
	"github.com/iotaledger/inx-api-core-v0/components/prometheus"
)

var (
	// Name of the app.
	Name = "inx-api-core-v0"

	// Version of the app.
	Version = "1.0.0-rc.4"
)

func App() *app.App {
	return app.New(Name, Version,
		app.WithInitComponent(InitComponent),
		app.WithComponents(
			shutdown.Component,
			database.Component,
			coreapi.Component,
			inx.Component,
			profiling.Component,
			prometheus.Component,
		),
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
