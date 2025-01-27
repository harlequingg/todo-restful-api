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
	"slices"
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

func (app *application) createTaskHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Content *string `json:"content"`
	}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}
	v := newValidator()
	v.checkCond(input.Content != nil, "content", "must be provided")
	v.checkCond(input.Content != nil && *input.Content != "", "content", "content must not be empty")
	if v.hasErrors() {
		writeError(w, v.toError(), http.StatusBadRequest)
		return
	}
	user := getUserFromRequest(r)
	if user == nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}
	t := &task{
		Content: *input.Content,
		UserID:  user.ID,
	}
	err = app.storage.insertTask(user, t)
	if err != nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"task": t}, http.StatusCreated)
}

func (app *application) updateTaskHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < -1 {
		writeError(w, errors.New("route paramter {id}: must to be a positive integer"), http.StatusBadRequest)
		return
	}

	var input struct {
		Content     *string `json:"content"`
		IsCompleted *bool   `json:"is_completed"`
	}

	err = json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}

	v := newValidator()
	if input.Content != nil {
		v.checkCond(*input.Content != "", "content", "must not be empty")
	}
	v.checkCond(input.Content != nil || input.IsCompleted != nil, "content or is_completed", "must be provided")
	if v.hasErrors() {
		writeError(w, v.toError(), http.StatusBadRequest)
		return
	}

	user := getUserFromRequest(r)
	if user == nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}

	t, err := app.storage.getTaskByID(id)
	if err != nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}
	if t == nil {
		writeError(w, errors.New("resource doesn't exist"), http.StatusNotFound)
		return
	}
	if t.UserID != user.ID {
		writeError(w, errors.New("access denied"), http.StatusConflict)
		return
	}
	if input.Content != nil {
		t.Content = *input.Content
	}
	if input.IsCompleted != nil {
		t.IsCompleted = *input.IsCompleted
	}
	err = app.storage.updateTask(t)
	if err != nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"task": t}, http.StatusOK)
}

func (app *application) getTaskHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < -1 {
		writeError(w, errors.New("route paramter {id}: must to be a positive integer"), http.StatusBadRequest)
		return
	}

	user := getUserFromRequest(r)
	if user == nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}

	t, err := app.storage.getTaskByID(id)
	if err != nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}
	if t == nil {
		writeError(w, errors.New("resource doesn't exist"), http.StatusNotFound)
		return
	}
	if t.UserID != user.ID {
		writeError(w, errors.New("access denied"), http.StatusConflict)
		return
	}
	writeJSON(w, map[string]any{"task": t}, http.StatusOK)
}

func (app *application) getTasksHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromRequest(r)
	if user == nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}

	query := r.URL.Query()
	sort := query.Get("sort")
	if sort == "" {
		sort = "id"
	}

	page := 1
	pageSize := 20

	pageStr := query.Get("page")
	if pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil || p <= 0 {
			writeError(w, errors.New(`invalid query parameter "page": must be a positive integer`), http.StatusBadRequest)
			return
		}
		page = p
	}
	pageSizeStr := query.Get("page_size")
	if pageSizeStr != "" {
		size, err := strconv.Atoi(pageSizeStr)
		if err != nil || size <= 0 {
			writeError(w, errors.New(`invalid query param "page_size": must be a positive integer`), http.StatusBadRequest)
			return
		}
		pageSize = size
	}

	v := newValidator()
	sortList := []string{"id", "-id", "created_at", "-created_at", "is_completed", "-is_completed"}
	v.checkCond(slices.Index(sortList, sort) != -1, "sort", fmt.Sprintf("must be one of the values %v", sortList))
	v.checkCond(page >= 1 && page <= 10_000_000, "page", "must be between 1 and 10_000_000")
	v.checkCond(pageSize >= 1 && page <= 100, "page_size", "must be between 1 and 100")

	content := query.Get("content")

	tasks, total, err := app.storage.getTasksForUser(user, sort, page, pageSize, content)
	if err != nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}
	if tasks == nil {
		writeError(w, errors.New("resource doesn't exist"), http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]any{"tasks": tasks, "total": total}, http.StatusOK)
}

func (app *application) deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < -1 {
		writeError(w, errors.New("route paramter {id}: must to be a positive integer"), http.StatusBadRequest)
		return
	}

	user := getUserFromRequest(r)
	if user == nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}

	t, err := app.storage.getTaskByID(id)
	if err != nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}
	if t == nil {
		writeError(w, errors.New("resource doesn't exist"), http.StatusNotFound)
		return
	}
	if t.UserID != user.ID {
		writeError(w, errors.New("access denied"), http.StatusConflict)
		return
	}
	err = app.storage.deleteTask(t)
	if err != nil {
		writeError(w, errors.New("internal server error"), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"message": "task deleted successfully"}, http.StatusOK)
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
