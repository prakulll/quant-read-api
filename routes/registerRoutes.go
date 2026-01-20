package routes

import (
	"net/http"
	"quant-read-api/controllers"
)

type apiRoute struct {
	Path    string
	Method  string
	Handler http.HandlerFunc
}

func RegisterRoutes() []apiRoute {
	return []apiRoute{

		{
			Path:    "/options/expiries",
			Method:  "GET",
			Handler: controllers.GetOptionExpiries,
		},

		{
			Path:    "/options/snapshot",
			Method:  "GET",
			Handler: controllers.GetOptionSnapshots,
		},

		{
			Path:    "/options/contract",
			Method:  "GET",
			Handler: controllers.GetOptionContract,
		},

		{
			Path:    "/options/contracts/by-premium",
			Method:  "GET",
			Handler: controllers.GetOptionContractsByPremium,
		},

		{
			Path:    "/index/data",
			Method:  "GET",
			Handler: controllers.GetIndexData,
		},

		{
			Path:    "/futures/data",
			Method:  "GET",
			Handler: controllers.GetFuturesData,
		},
	}
}
