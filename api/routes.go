package main

import (
	"net/http"
)

func composeRoutes(app *application) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/healthcheck", app.healthCheckHandler)

	mux.HandleFunc("POST /v1/users/{id}/activation", app.sendActivationCodeHandler)
	mux.HandleFunc("PUT /v1/users/{id}/activation", app.activateUserHandler)
	mux.HandleFunc("POST /v1/users/authentication", app.authenticateUserHandler)

	mux.HandleFunc("POST /v1/users", app.createUserHandler)
	mux.HandleFunc("PUT /v1/users", app.requireAuthenticatedUser(requireActivatedUser(app.updateUserHandler)))
	mux.HandleFunc("GET /v1/users", app.requireAuthenticatedUser(requireActivatedUser(app.getUserHandler)))
	mux.HandleFunc("DELETE /v1/users", app.requireAuthenticatedUser(requireActivatedUser(app.deleteUserHandler)))

	mux.HandleFunc("POST /v1/tasks", app.requireAuthenticatedUser(requireActivatedUser(app.createTaskHandler)))
	mux.HandleFunc("PUT /v1/tasks/{id}", app.requireAuthenticatedUser(requireActivatedUser(app.updateTaskHandler)))
	mux.HandleFunc("GET /v1/tasks", app.requireAuthenticatedUser(requireActivatedUser(app.getTasksHandler)))
	mux.HandleFunc("GET /v1/tasks/{id}", app.requireAuthenticatedUser(requireActivatedUser(app.getTaskHandler)))
	mux.HandleFunc("DELETE /v1/tasks/{id}", app.requireAuthenticatedUser(requireActivatedUser(app.deleteTaskHandler)))

	if app.config.limiter.enabled {
		return app.enableCORS(app.rateLimit(mux))
	}
	return app.enableCORS(mux)
}
