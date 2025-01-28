package main

import (
	"net/http"
)

func composeRoutes(app *application) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/healthcheck", app.healthCheckHandler)

	mux.HandleFunc("POST /v1/users", app.requireAuth(requireUserActivation(app.createUserHandler)))
	mux.HandleFunc("PUT /v1/users/{id}", app.requireAuth(requireUserActivation(app.updateUserHandler)))
	mux.HandleFunc("GET /v1/users/{id}", app.requireAuth(requireUserActivation(app.getUserHandler)))
	mux.HandleFunc("DELETE /v1/users/{id}", app.requireAuth(requireUserActivation(app.deleteUserHandler)))

	mux.HandleFunc("POST /v1/tasks", app.requireAuth(requireUserActivation(app.createTaskHandler)))
	mux.HandleFunc("PUT /v1/tasks/{id}", app.requireAuth(requireUserActivation(app.updateTaskHandler)))
	mux.HandleFunc("GET /v1/tasks", app.requireAuth(requireUserActivation(app.getTasksHandler)))
	mux.HandleFunc("GET /v1/tasks/{id}", app.requireAuth(requireUserActivation(app.getTaskHandler)))
	mux.HandleFunc("DELETE /v1/tasks/{id}", app.requireAuth(requireUserActivation(app.deleteTaskHandler)))

	mux.HandleFunc("POST /v1/users/{id}/activate", app.sendActivationCodeHandler)
	mux.HandleFunc("PUT /v1/users/{id}/activate", app.activateUserHandler)
	mux.HandleFunc("POST /v1/users/auth", app.authenticateUserHandler)
	if app.config.limiter.enabled {
		return app.rateLimit(mux)
	}
	return mux
}
