package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func composeRoutes(app *application) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/healthcheck", app.healthCheckHandler)

	mux.HandleFunc("GET /v1/users/{id}", app.authenticate(app.getUserHandler))
	mux.HandleFunc("POST /v1/users", app.authenticate(app.createUserHandler))
	mux.HandleFunc("PUT /v1/users/{id}", app.authenticate(app.updateUserHandler))
	mux.HandleFunc("DELETE /v1/users/{id}", app.authenticate(app.deleteUserHandler))

	mux.HandleFunc("POST /v1/users/{id}/activate", app.sendActivationCodeHandler)
	mux.HandleFunc("PUT /v1/users/{id}/activate", app.activateUserHandler)
	mux.HandleFunc("POST /v1/users/auth", app.authenticateUserHandler)

	return mux
}

func (app *application) authenticate(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			return
		}
		parts := strings.Fields(authHeader)
		if len(parts) != 2 || parts[0] != "Bearer" {
			writeError(w, errors.New("invalid Authorization header"), http.StatusUnauthorized)
			return
		}
		tokenStr := parts[1]
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(app.config.jwtSecret), nil
		})
		if err != nil {
			log.Println(err)
			writeError(w, errors.New("invalid token"), http.StatusUnauthorized)
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			log.Println(err)
			writeError(w, errors.New("invalid token"), http.StatusUnauthorized)
			return
		}
		userID := int(claims["user_id"].(float64))
		expiresAtStr := claims["expires_at"].(string)
		expiresAt, err := time.Parse(time.RFC822, expiresAtStr)
		if err != nil {
			log.Println(err)
			writeError(w, errors.New("invalid token"), http.StatusUnauthorized)
			return
		}

		if time.Now().After(expiresAt) {
			writeError(w, errors.New("invalid token"), http.StatusUnauthorized)
			return
		}
		u, err := app.storage.getUserByID(userID)
		if err != nil {
			log.Println(err)
			writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
			return
		}
		// TODO: add user to request content
		log.Println(u)
		next.ServeHTTP(w, r)
	})
}
