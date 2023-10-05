package coreapi

import (
	"github.com/iotaledger/hive.go/app"
)

// ParametersRestAPI contains the definition of the parameters used by the legacy API HTTP server.
type ParametersRestAPI struct {
	// BindAddress defines the bind address on which the legacy API HTTP server listens.
	BindAddress string `default:"localhost:9093" usage:"the bind address on which the legacy API HTTP server listens"`

	// AdvertiseAddress defines the address of the legacy API HTTP server which is advertised to the INX Server (optional).
	AdvertiseAddress string `default:"" usage:"the address of the legacy API HTTP server which is advertised to the INX Server (optional)"`

	Limits struct {
		// the maximum number of characters that the body of an API call may contain
		MaxBodyLength string `default:"1M" usage:"the maximum number of characters that the body of an API call may contain"`
		// the maximum number of results that may be returned by an endpoint
		MaxResults int `default:"1000" usage:"the maximum number of results that may be returned by an endpoint"`
	}

	// SwaggerEnabled defines whether to provide swagger API documentation under endpoint "/swagger"
	SwaggerEnabled bool `default:"false" usage:"whether to provide swagger API documentation under endpoint \"/swagger\""`

	// UseGZIP defines whether to use the gzip middleware to compress HTTP responses
	UseGZIP bool `default:"true" usage:"use the gzip middleware to compress HTTP responses"`

	// DebugRequestLoggerEnabled defines whether the debug logging for requests should be enabled
	DebugRequestLoggerEnabled bool `default:"false" usage:"whether the debug logging for requests should be enabled"`
}

var ParamsRestAPI = &ParametersRestAPI{}

var params = &app.ComponentParams{
	Params: map[string]any{
		"restAPI": ParamsRestAPI,
	},
	Masked: nil,
}
