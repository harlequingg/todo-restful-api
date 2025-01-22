package main

import "net/http"

func composeRoutes(app *application) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/healthcheck", app.healthCheckHandler)
	return mux
}
