package main

import "time"

type user struct {
	ID           int       `json:"id"`
	CreateAt     time.Time `json:"created_at"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash []byte    `json:"-"`
	IsActivated  bool      `json:"is_activated"`
	Version      int       `json:"-"`
}
