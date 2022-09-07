package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-redis/redis/v9"
	"github.com/rog-golang-buddies/rapidmidiex"
	"github.com/rog-golang-buddies/rapidmidiex/api"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/spf13/viper"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	err := rmx.LoadConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v", err.Error())
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=true",
		viper.GetString("DB_USER"),
		viper.GetString("DB_PASSWORD"),
		viper.GetString("DB_HOST"),
		viper.GetString("DB_PORT"),
		viper.GetString("DB_NAME"),
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalln(err)
	}

	// run db migrations
	migrations := &migrate.FileMigrationSource{
		Dir: "api/internal/db/migration",
	}

	_, err = migrate.Exec(db, "mysql", migrations, migrate.Up)
	if err != nil {
		log.Fatalln(err)
	}

	rtdb := redis.NewClient(&redis.Options{
		Addr:     viper.GetString("REDIS_ADDR"),
		Password: viper.GetString("REDIS_PASSWORD"),
		DB:       0,
	})
	cidb := redis.NewClient(&redis.Options{
		Addr:     viper.GetString("REDIS_ADDR"),
		Password: viper.GetString("REDIS_PASSWORD"),
		DB:       1,
	})
	ptdb := redis.NewClient(&redis.Options{
		Addr:     viper.GetString("REDIS_ADDR"),
		Password: viper.GetString("REDIS_PASSWORD"),
		DB:       2,
	})

	authService := api.AuthService{
		DBCon:                db,
		RedisRefreshTokenDB:  rtdb,
		RedisClientIDDB:      cidb,
		RedisPasswordTokenDB: ptdb,
	}
	jamService := api.JamService{
		DBCon: db,
	}

	server := api.Server{
		Port:   ":8080",
		Router: chi.NewMux(),
	}

	server.Router.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Logger)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authService.Register)
			r.Post("/login", authService.Login)
			r.Get("/refresh_token", authService.RefreshToken)
			r.Get("/logout", authService.Logout)
		})
		r.Route("/jam", func(r chi.Router) {
			r.Use(authService.CheckAuth)
			r.Post("/new", jamService.NewSession)
			r.Get("/{session_id}/join", jamService.JoinSession)
		})
		r.Route("/users", func(r chi.Router) {
			r.Use(authService.CheckAuth)
			r.Get("/me", authService.GetUserInfo)
			r.Patch("/me", authService.UpdateUserInfo)
		})
	})

	log.Println("starting the server")
	server.ServeHTTP()
}
