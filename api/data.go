package main

import "time"

type user struct {
	ID           int       `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash []byte    `json:"-"`
	IsActivated  bool      `json:"is_activated"`
	Version      int       `json:"-"`
}

type task struct {
	ID          int       `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UserID      int       `json:"user_id"`
	Content     string    `json:"content"`
	IsCompleted bool      `json:"is_completed"`
	Version     int       `json:"-"`
}
