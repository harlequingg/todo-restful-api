package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
}

type application struct {
	config config
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	var cfg config
	flag.IntVar(&cfg.port, "port", 3000, "Server Port")
	flag.StringVar(&cfg.env, "evn", "development", "Environment [development|production]")
	flag.Parse()

	app := &application{
		config: cfg,
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      composeRoutes(app),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	log.Printf("Starting %s server on port %d\n", cfg.env, cfg.port)
	err := srv.ListenAndServe()
	log.Fatal(err)
}
