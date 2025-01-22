package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
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

	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(cfg.db.maxOpenConnections)
	db.SetMaxIdleConns(cfg.db.maxIdelConnections)
	db.SetConnMaxIdleTime(cfg.db.maxIdelTime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("established a connection with database")

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
	err = srv.ListenAndServe()
	log.Fatal(err)
}
