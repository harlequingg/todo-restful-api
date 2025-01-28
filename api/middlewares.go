package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/time/rate"
)

func (app *application) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
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
			return []byte(app.config.jwt.secret), nil
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
		if u == nil {
			writeError(w, errors.New("user no longer exists"), http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := getUserFromRequest(r)
		if !user.IsActivated {
			writeError(w, errors.New("your user account must be activated to access this resource"), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func (app *application) rateLimit(next http.Handler) http.HandlerFunc {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}
	var (
		mu      sync.RWMutex
		clients = make(map[string]client)
	)
	go func() {
		for {
			time.Sleep(time.Minute)
			func() {
				mu.Lock()
				defer mu.Unlock()
				for ip, client := range clients {
					if time.Since(client.lastSeen) >= time.Minute*3 {
						delete(clients, ip)
					}
				}
			}()
		}
	}()
	return func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			log.Println(err)
			writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
			return
		}
		mu.Lock()
		c, ok := clients[ip]
		if !ok {
			c = client{
				limiter: rate.NewLimiter(rate.Limit(app.config.limiter.maxRequestPerSecond), app.config.limiter.burst),
			}
		}
		c.lastSeen = time.Now()
		clients[ip] = c
		if !c.limiter.Allow() {
			mu.Unlock()
			writeError(w, errors.New("rate limit exceeded"), http.StatusTooManyRequests)
			return
		}
		mu.Unlock()
		next.ServeHTTP(w, r)
	}
}

func (app *application) enableCORS(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		w.Header().Add("Vary", "Access-Control-Request-Method")

		origin := w.Header().Get("Origin")
		if origin != "" {
			for _, o := range app.config.cors.trustedOrigins {
				if origin == o || o == "*" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					// preflight request
					if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
						w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
						w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
						w.WriteHeader(http.StatusOK)
						return
					}
					break
				}
			}
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
