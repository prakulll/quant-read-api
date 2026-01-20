package routes

import "github.com/gorilla/mux"

func ServeRoutes(router *mux.Router) {
	api := router.PathPrefix("/api/v1").Subrouter()
	for _, r := range RegisterRoutes() {
		api.HandleFunc(r.Path, r.Handler).Methods(r.Method)
	}
}
