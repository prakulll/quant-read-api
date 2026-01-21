package routes

import "github.com/gorilla/mux"

func ServeRoutes(router *mux.Router) {

	v1 := router.PathPrefix("/api/v1").Subrouter()
	v2 := router.PathPrefix("/api/v2").Subrouter()

	for _, r := range RegisterRoutes() {
		switch r.Version {
		case "v1":
			v1.HandleFunc(r.Path, r.Handler).Methods(r.Method)
		case "v2":
			v2.HandleFunc(r.Path, r.Handler).Methods(r.Method)
		}
	}
}
