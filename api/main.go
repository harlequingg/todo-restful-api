package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	db   struct {
		dsn                string
		maxOpenConnections int
		maxIdelConnections int
		maxIdelTime        time.Duration
	}
}

type application struct {
	config config
	db     *sql.DB
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	var cfg config
	flag.IntVar(&cfg.port, "port", 3000, "Server Port")
	flag.StringVar(&cfg.env, "evn", "development", "Environment [development|production]")

	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("TODO_DB_DSN"), "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConnections, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdelConnections, "db-max-idel-conns", 25, "PostgreSQL max idel connections")
	var maxIdelTime string
	flag.StringVar(&maxIdelTime, "db-max-idel-time", "15m", "PostgreSQL max connection idel time")
	flag.Parse()

	d, err := time.ParseDuration(maxIdelTime)
	if err != nil {
		cfg.db.maxIdelTime = 15 * time.Minute
		log.Printf(`invalid value %s for flag "db-max-idel-time" defaulting to %s`, maxIdelTime, cfg.db.maxIdelTime)
	} else {
		cfg.db.maxIdelTime = d
	}

	db, err := openDB(cfg)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("established a connection with database")

	app := &application{
		config: cfg,
		db:     db,
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      composeRoutes(app),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	log.Printf("Starting %s server on port %d\n", cfg.env, cfg.port)
	err = srv.ListenAndServe()
	log.Fatal(err)
}
