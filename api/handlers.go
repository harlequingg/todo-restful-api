package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

func (app *application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-type", "application/json")
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
		http.Error(w, composeInternalServerError(), http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

func composeJsonError(err error) string {
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

func composeInternalServerError() string {
	return composeJsonError(errors.New("internal server error"))
}
