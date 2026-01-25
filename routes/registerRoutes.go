package routes

import (
	"net/http"
	"quant-read-api/controllers"
)

type apiRoute struct {
	Version string // "v1" or "v2"
	Path    string
	Method  string
	Handler http.HandlerFunc
}

func RegisterRoutes() []apiRoute {
	return []apiRoute{

		// ---------- V1 ----------
		{
			Version: "v1",
			Path:    "/index/data",
			Method:  "GET",
			Handler: controllers.GetIndexData,
		},
		{
			Version: "v1",
			Path:    "/futures/data",
			Method:  "GET",
			Handler: controllers.GetFuturesData,
		},
		{
			Version: "v1",
			Path:    "/options/expiries",
			Method:  "GET",
			Handler: controllers.GetOptionExpiries,
		},
		{
			Version: "v1",
			Path:    "/options/snapshot",
			Method:  "GET",
			Handler: controllers.GetOptionSnapshots,
		},
		{
			Version: "v1",
			Path:    "/options/contract",
			Method:  "GET",
			Handler: controllers.GetOptionContract,
		},
		{
			Version: "v1",
			Path:    "/options/contracts/by-premium",
			Method:  "GET",
			Handler: controllers.GetOptionContractsByPremium,
		},

		// ---------- V2 ----------
		{
			Version: "v2",
			Path:    "/index/data",
			Method:  "GET",
			Handler: controllers.GetIndexDataV2,
		},

		{
			Version: "v2",
			Path:    "/futures/data",
			Method:  "GET",
			Handler: controllers.GetFuturesDataV2,
		},

		{
			Version: "v2",
			Path:    "/options/snapshot",
			Method:  "GET",
			Handler: controllers.GetOptionSnapshotsV2,
		},
	}
}
