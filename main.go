package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/nitishm/go-rejson/v4"
	"github.com/rs/cors"
)

var ctx = context.Background()

func createRedisClient(host string, port string) (*redis.Client, *rejson.Handler) {
	rh := rejson.NewReJSONHandler()
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: "",
		DB:       0,
	})
	rh.SetGoRedisClient(rdb)
	return rdb, rh
}

func setupHttpServer() {
	host, dbPort := "redis", os.Getenv("DB_PORT")
	rdb, rdbJson := createRedisClient(host, dbPort)
	mux := mux.NewRouter()

	mux.HandleFunc("/{key}/json", jsonGet(rdbJson)).Methods("GET")
	mux.HandleFunc("/{key}/json", jsonSet(rdbJson)).Methods("PUT", "POST")
	mux.HandleFunc("/{key}/json", delete(rdb)).Methods("DELETE")

	mux.HandleFunc("/{key}/pop", jsonArrayPop(rdbJson)).Methods("GET")
	mux.HandleFunc("/{key}/insert", jsonArrayInsert(rdbJson)).Methods("PUT", "POST")

	mux.HandleFunc("/{key}", get(rdb)).Methods("GET")
	mux.HandleFunc("/{key}", set(rdb)).Methods("PUT", "POST")
	mux.HandleFunc("/{key}", delete(rdb)).Methods("DELETE")

	apiPort := os.Getenv("API_PORT")
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "PUT", "POST", "DELETE"},
		AllowCredentials: true,
	})
	handler := c.Handler(mux)
	http.ListenAndServeTLS(fmt.Sprintf(":%s", apiPort), "server.crt", "server.key", handler)
}

func main() {
	setupHttpServer()
}
