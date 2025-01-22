package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

func (app *application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
	heathCheck := struct {
		Status      string `json:"status"`
		Environment string `json:"environment"`
		Version     string `json:"version"`
	}{
		Status:      "available",
		Environment: app.config.env,
		Version:     version,
	}
	data, err := json.Marshal(heathCheck)
	if err != nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

func (app *application) createUserHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
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
	h := w.Header()
	h.Del("Content-Length")
	h.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(statusCode)
	fmt.Fprintln(w, composeJSONError(err))
}
