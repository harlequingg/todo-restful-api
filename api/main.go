package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
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
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	jwt struct {
		secret string
	}
	limiter struct {
		maxRequestPerSecond float64
		burst               int
		enabled             bool
	}
	cors struct {
		trustedOrigins []string
	}
}

type application struct {
	config  config
	storage *storage
	mailer  *mailer
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	var cfg config
	flag.IntVar(&cfg.port, "port", 3000, "Server Port")
	flag.StringVar(&cfg.env, "evn", "development", "Environment [development|production]")

	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("DB_DSN"), "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConnections, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdelConnections, "db-max-idel-conns", 25, "PostgreSQL max idel connections")
	var maxIdelTime string
	flag.StringVar(&maxIdelTime, "db-max-idel-time", "15m", "PostgreSQL max connection idel time")

	flag.StringVar(&cfg.smtp.host, "smtp-host", os.Getenv("SMTP_HOST"), "SMTP host")

	smtpPort, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		log.Fatal(err)
	}

	flag.IntVar(&cfg.smtp.port, "smtp-port", smtpPort, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", os.Getenv("SMTP_USERNAME"), "SMTP host")
	flag.StringVar(&cfg.smtp.password, "smtp-password", os.Getenv("SMTP_PASSWORD"), "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", os.Getenv("SMTP_SENDER"), "SMTP sender")

	flag.StringVar(&cfg.jwt.secret, "jwt-secret", os.Getenv("JWT_SECRET"), "JWT secret")

	flag.Float64Var(&cfg.limiter.maxRequestPerSecond, "limiter-max-rps", 2, "Rate Limiter max requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate Limiter max burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	var trustedOrigins string
	flag.StringVar(&trustedOrigins, "cors-trusted-origins", "*", "Trusted CORS origins saperated by space")
	flag.Parse()

	d, err := time.ParseDuration(maxIdelTime)
	if err != nil {
		cfg.db.maxIdelTime = 15 * time.Minute
		log.Printf(`invalid value %s for flag "db-max-idel-time" defaulting to %s`, maxIdelTime, cfg.db.maxIdelTime)
	} else {
		cfg.db.maxIdelTime = d
	}

	cfg.cors.trustedOrigins = strings.Fields(trustedOrigins)

	db, err := openDB(cfg)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("established a connection with database")

	if cfg.jwt.secret == "" {
		secret := make([]byte, 32)
		_, err = rand.Read(secret[:])
		if err != nil {
			log.Fatal(err)
		}
		cfg.jwt.secret = string(secret)
	}

	app := &application{
		config:  cfg,
		storage: newStorage(db),
		mailer:  newMailer(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
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
