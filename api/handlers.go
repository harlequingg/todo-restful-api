package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

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

	writeJSON(w, u, http.StatusCreated)
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
