package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

//go:embed templates
var templates embed.FS

func (app *application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	heathCheck := struct {
		Status      string `json:"status"`
		Environment string `json:"environment"`
		Version     string `json:"version"`
	}{
		Status:      "available",
		Environment: app.config.env,
		Version:     version,
	}
	writeJSON(w, heathCheck, http.StatusOK)
}

func (app *application) createUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}

	v := newValidator()
	v.checkCond(input.Name != "", "name", "must be provided")
	v.checkCond(len(input.Name) <= 255, "name", "must be atmost 255 characters")
	v.checkEmail(input.Email)
	v.checkPassword(input.Password)
	if v.hasErrors() {
		writeError(w, v.toError(), http.StatusBadRequest)
		return
	}

	u, err := app.storage.getUserByEmail(input.Email)
	if err != nil {
		log.Println(err)
		writeError(w, errors.New("internal server errors"), http.StatusInternalServerError)
		return
	}

	if u != nil {
		writeError(w, errors.New("user already exists"), http.StatusConflict)
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 13)
	if err != nil {
		log.Println(err)
		writeError(w, err, http.StatusConflict)
		return
	}

	u = &user{
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: passwordHash,
	}
	err = app.storage.insertUser(u)
	if err != nil {
		log.Println(err)
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFS(templates, "templates/*.gotmpl")
	if err != nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}
	code := uint16(rand.Uint())
	err = app.mailer.send(u.Email, tmpl, map[string]any{"code": code})
	if err != nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}
	app.storage.useractivationCache.Set(u, code, time.Minute)
	writeJSON(w, map[string]any{"user": u, "message": fmt.Sprintf("we have sent an activation code to your email: %s", u.Email)}, http.StatusCreated)
}

func (app *application) getUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < -1 {
		writeError(w, errors.New("route paramter {id}: must to be a positive integer"), http.StatusBadRequest)
		return
	}

	u, err := app.storage.getUserByID(id)
	if err != nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}
	if u == nil {
		writeError(w, errors.New("user doesn't exist"), http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]any{"user": u}, http.StatusOK)
}

func (app *application) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < -1 {
		writeError(w, errors.New("route paramter {id}: must to be a positive integer"), http.StatusBadRequest)
		return
	}

	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err = json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}

	u, err := app.storage.getUserByID(id)
	if err != nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}
	if u == nil {
		writeError(w, errors.New("user doesn't exist with the provided id"), http.StatusNotFound)
		return
	}

	v := newValidator()

	if input.Name != "" {
		v.checkCond(input.Name != "", "name", "must be provided")
		v.checkCond(len(input.Name) <= 255, "name", "must be atmost 255 characters")
	}

	if input.Email != "" {
		v.checkEmail(input.Email)
	}

	if input.Password != "" {
		v.checkPassword(input.Password)
	}

	v.checkCond(input.Name != "" || input.Email != "" || input.Password != "", "name or email or password", "must be provided")

	if v.hasErrors() {
		writeError(w, v.toError(), http.StatusBadRequest)
		return
	}

	if input.Name != "" {
		u.Name = input.Name
	}
	if input.Email != "" {
		u.Email = input.Email
	}
	if input.Password != "" {
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 13)
		if err != nil {
			log.Println(err)
			writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
			return
		}
		u.PasswordHash = passwordHash
	}

	err = app.storage.updateUser(u)
	if err != nil {
		log.Println(err)
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{"user": u}, http.StatusOK)
}

func (app *application) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < -1 {
		writeError(w, errors.New("route paramter {id}: must to be a positive integer"), http.StatusBadRequest)
		return
	}

	u, err := app.storage.getUserByID(id)
	if err != nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}

	if u == nil {
		writeError(w, errors.New("user doesn't exist"), http.StatusNotFound)
		return
	}

	err = app.storage.deleteUser(u)
	if err != nil {
		log.Println(err)
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"message": "user successfully deleted"}, http.StatusOK)
}

func (app *application) sendActivationCodeHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < -1 {
		writeError(w, errors.New("route paramter {id}: must to be a positive integer"), http.StatusBadRequest)
		return
	}

	u, err := app.storage.getUserByID(id)
	if err != nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}

	if u.IsActivated {
		writeJSON(w, map[string]any{"message": "user already activated"}, http.StatusConflict)
		return
	}

	if app.storage.useractivationCache.HasExpired(u) {
		tmpl, err := template.ParseFS(templates, "templates/*.gotmpl")
		if err != nil {
			writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
			return
		}
		code := uint16(rand.Uint())
		err = app.mailer.send(u.Email, tmpl, map[string]any{"code": code})
		if err != nil {
			log.Println(err)
			writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
			return
		}
		app.storage.useractivationCache.Set(u, code, time.Minute)
	}
	writeJSON(w, map[string]any{"message": fmt.Sprintf("we have sent an activation code to your email: %s", u.Email)}, http.StatusOK)
}

func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < -1 {
		writeError(w, errors.New("route paramter {id}: must to be a positive integer"), http.StatusBadRequest)
		return
	}

	var input struct {
		Code *int `json:"code"`
	}

	err = json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}

	if input.Code == nil {
		writeError(w, errors.New("code must be provided in request body"), http.StatusBadRequest)
		return
	}

	u, err := app.storage.getUserByID(id)
	if err != nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}
	if u.IsActivated {
		writeJSON(w, map[string]any{"message": "user already activated"}, http.StatusConflict)
		return
	}
	activationCode, expired := app.storage.useractivationCache.Get(u)
	if expired {
		writeJSON(w, map[string]any{"message": "code has expired"}, http.StatusConflict)
		return
	}
	if activationCode != *input.Code {
		writeJSON(w, map[string]any{"message": "invalid activation code"}, http.StatusConflict)
		return
	}
	u.IsActivated = true
	err = app.storage.updateUser(u)
	if err != nil {
		log.Println(err)
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"user": u}, http.StatusOK)
}

func (app *application) authenticateUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}

	v := newValidator()
	v.checkEmail(input.Email)
	v.checkPassword(input.Password)

	if v.hasErrors() {
		writeError(w, v.toError(), http.StatusBadRequest)
		return
	}

	u, err := app.storage.getUserByEmail(input.Email)
	if err != nil {
		log.Println(err)
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}

	if u == nil {
		writeError(w, errors.New("email or password are not correct"), http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(input.Password))
	if err != nil {
		writeError(w, errors.New("email or password are not correct"), http.StatusUnauthorized)
		return
	}

	claims := jwt.MapClaims{
		"user_id":    u.ID,
		"expires_at": time.Now().Add(24 * time.Hour).Format(time.RFC822),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(app.config.jwtSecret))
	if err != nil {
		log.Println(err)
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"token": tokenStr}, http.StatusCreated)
}

func composeJSONError(err error) string {
	jsonError := map[string]string{
		"error": err.Error(),
	}
	result, err := json.Marshal(jsonError)
	if err != nil {
		log.Println(err)
		return ""
	}
	return string(result)
}

func writeError(w http.ResponseWriter, err error, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	fmt.Fprintln(w, composeJSONError(err))
}

func writeJSON(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	j, err := json.Marshal(data)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(statusCode)
	w.Write(j)
}
