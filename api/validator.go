package main

import (
	"encoding/json"
	"errors"
	"regexp"
)

var emailRegexp = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

type validator struct {
	errors map[string]string
}

func newValidator() *validator {
	return &validator{
		errors: make(map[string]string),
	}
}

func (v *validator) toError() error {
	if v == nil {
		return errors.New("")
	}
	data, err := json.Marshal(v.errors)
	if err != nil {
		return err
	}
	return errors.New(string(data))
}

func (v *validator) hasErrors() bool {
	return len(v.errors) != 0
}

func (v *validator) checkCond(cond bool, key, msg string) {
	if cond {
		return
	}
	if _, ok := v.errors[key]; !ok {
		v.errors[key] = msg
	}
}

func (v *validator) checkEmail(email string) {
	v.checkCond(email != "", "email", "must be provided")
	v.checkCond(emailRegexp.Match([]byte(email)), "email", "must be a valid email address")
}

func (v *validator) checkPassword(password string) {
	v.checkCond(password != "", "password", "must be provided")
	v.checkCond(len(password) >= 8, "password", "must be atleast 8 characters long")
	v.checkCond(len(password) <= 72, "password", "must be atmost 72 characters long")
}
