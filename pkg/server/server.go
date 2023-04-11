package server

import (
	"github.com/labstack/echo/v4"
	"github.com/pangpanglabs/echoswagger/v2"

	"github.com/iotaledger/hive.go/app"
	"github.com/iotaledger/inx-api-core-v0/pkg/database"
)

const (
	APIRoute = "/api/core/v0"
)

type DatabaseServer struct {
	AppInfo                 *app.Info
	Database                *database.Database
	RestAPILimitsMaxResults int
	RPCEndpoints            map[string]rpcEndpoint
}

func NewDatabaseServer(swagger echoswagger.ApiRoot, appInfo *app.Info, db *database.Database, maxResults int) *DatabaseServer {
	s := &DatabaseServer{
		AppInfo:                 appInfo,
		Database:                db,
		RestAPILimitsMaxResults: maxResults,
		RPCEndpoints:            make(map[string]rpcEndpoint),
	}

	s.configureRoutes(swagger.Group("root", APIRoute))

	return s
}

func CreateEchoSwagger(e *echo.Echo, version string, enabled bool) echoswagger.ApiRoot {
	if !enabled {
		return echoswagger.NewNop(e)
	}

	echoSwagger := echoswagger.New(e, "/swagger", &echoswagger.Info{
		Title:       "inx-api-core-v0 API",
		Description: "REST/RPC API for IOTA legacy",
		Version:     version,
	})

	echoSwagger.SetExternalDocs("Find out more about inx-api-core-v0", "https://wiki.iota.org/shimmer/inx-api-core-v0/welcome/")
	echoSwagger.SetUI(echoswagger.UISetting{DetachSpec: false, HideTop: false})
	echoSwagger.SetScheme("http", "https")
	echoSwagger.SetRequestContentType(echo.MIMEApplicationJSON)
	echoSwagger.SetResponseContentType(echo.MIMEApplicationJSON)

	return echoSwagger
}
