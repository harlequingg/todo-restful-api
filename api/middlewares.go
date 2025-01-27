package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
)

func (app *application) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, errors.New("invalid Authorization header"), http.StatusUnauthorized)
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
		ctx := context.WithValue(r.Context(), userContextKey, u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requireUserActivation(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := getUserFromRequest(r)
		if !user.IsActivated {
			writeError(w, errors.New("your user account must be activated to access this resource"), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}
}

type userContext string

const userContextKey userContext = "userContextKey"

func getUserFromRequest(r *http.Request) *user {
	u, _ := r.Context().Value(userContextKey).(*user)
	return u
}
