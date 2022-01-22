package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/nitishm/go-rejson/v4"
)

func apiError(msg string, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)

	log.Println(fmt.Sprintf("%s:%s", msg, err.Error()))
	resp := make(map[string]string)
	resp["Error"] = msg
	resp["ErrorMessage"] = err.Error()
	respJson, err := json.Marshal(resp)
	if err != nil {
		log.Fatalln("Error marshalling JSON")
	}
	w.Write(respJson)
}

// Basic redis commands
func get(rdb *redis.Client) func(http.ResponseWriter, *http.Request) {
	f := func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		log.Println(fmt.Sprintf("Request recieved for GET %s", vars["key"]))

		res, err := rdb.Get(ctx, vars["key"]).Result()
		if err == redis.Nil {
			log.Println("No value found in db")
			fmt.Fprintf(w, "")
		} else if err != nil {
			apiError("Unexpected error occured", err, w)
		} else {
			log.Println("Success")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, res)
		}
	}
	return f
}

func set(rdb *redis.Client) func(http.ResponseWriter, *http.Request) {
	f := func(w http.ResponseWriter, req *http.Request) {
		// Get body
		vars := mux.Vars(req)
		var ttl time.Duration
		if req.URL.Query().Has("ttl") {
			s, err := strconv.Atoi(req.URL.Query().Get("ttl"))
			if err != nil {
				apiError("Incompatible ttl received, must be an integer", err, w)
				return
			}
			ttl = time.Duration(s) * time.Second
		} else {
			ttl = 0
		}

		log.Println(fmt.Sprintf("Request recieved for SET %s with ttl of %s", vars["key"], ttl))
		body, err := io.ReadAll(req.Body)
		if err != nil {
			apiError("Error parsing body", err, w)
			return
		}

		// Set value
		err = rdb.Set(ctx, vars["key"], body, ttl).Err()
		if err != nil {
			apiError("Error setting value to redis", err, w)
			return
		}
		log.Println("Success")
		w.WriteHeader(http.StatusOK)
	}
	return f
}

func delete(rdb *redis.Client) func(http.ResponseWriter, *http.Request) {
	f := func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		log.Println(fmt.Sprintf("Request recieved for DELETE %s", vars["key"]))

		err := rdb.Del(ctx, vars["key"]).Err()
		if err != nil {
			apiError("Error deleting value from redis", err, w)
			return
		}
		log.Println("Success")
		w.WriteHeader(http.StatusOK)
	}
	return f
}

// RedisJson commands
func jsonGet(rdb *rejson.Handler) func(http.ResponseWriter, *http.Request) {
	f := func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		path := "."
		if req.URL.Query().Has("path") {
			path = req.URL.Query().Get("path")
		}

		log.Println(fmt.Sprintf("Request recieved for GET /json/%s with path %s", vars["key"], path))

		res, err := rdb.JSONGet(vars["key"], path)
		if err == redis.Nil {
			log.Println("No value found in db")
			fmt.Fprintf(w, "")
		} else if err != nil {
			apiError("Unexpected error occured", err, w)
		} else {
			log.Println("Success")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, string(res.([]byte)))
		}
	}
	return f
}

func jsonSet(rdb *rejson.Handler) func(http.ResponseWriter, *http.Request) {
	f := func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		path := "."
		if req.URL.Query().Has("path") {
			path = req.URL.Query().Get("path")
		}

		log.Println(fmt.Sprintf("Request recieved for SET /json/%s with path %s", vars["key"], path))

		decoder := json.NewDecoder(req.Body)
		var data interface{}
		err := decoder.Decode(&data)
		if err != nil {
			apiError("Unable to decode body", err, w)
			return
		}

		// Set
		res, err := rdb.JSONSet(vars["key"], path, data)
		if err != nil {
			apiError("Failed to set", err, w)
			return
		}
		if res == nil {
			apiError("Failed to set", errors.New("Unknown failure. Potentially trying to set property on non-json type"), w)
			return
		}
		log.Println("Success")
		w.WriteHeader(http.StatusOK)
	}
	return f
}

func jsonArrayPop(rdb *rejson.Handler) func(http.ResponseWriter, *http.Request) {
	f := func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		path := "."
		index := 0
		var err error
		if req.URL.Query().Has("path") {
			path = req.URL.Query().Get("path")
		}
		if req.URL.Query().Has("index") {
			index, err = strconv.Atoi(req.URL.Query().Get("index"))
			if err != nil {
				apiError("Incompatible index received, must be an integer", err, w)
			}
		}

		log.Println(fmt.Sprintf("Request recieved for POP /json/array/pop/%s with path %s and index %d", vars["key"], path, index))

		// Set
		res, err := rdb.JSONArrPop(vars["key"], path, index)
		if err != nil {
			apiError("Failed to pop", err, w)
			return
		} else {
			log.Println("Success")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, string(res.([]byte)))
		}
	}
	return f
}

func jsonArrayInsert(rdb *rejson.Handler) func(http.ResponseWriter, *http.Request) {
	f := func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		path := "."
		index := -1
		var err error
		if req.URL.Query().Has("path") {
			path = req.URL.Query().Get("path")
		}
		if req.URL.Query().Has("index") {
			index, err = strconv.Atoi(req.URL.Query().Get("index"))
			if err != nil {
				apiError("Incompatible index received, must be an integer", err, w)
			}
		}

		log.Println(fmt.Sprintf("Request recieved for INSERT /json/array/insert/%s with path %s and index %d", vars["key"], path, index))

		decoder := json.NewDecoder(req.Body)
		var data interface{}
		err = decoder.Decode(&data)
		if err != nil {
			apiError("Unable to decode body", err, w)
			return
		}

		if index == -1 {
			_, err = rdb.JSONArrAppend(vars["key"], path, data)

		} else {
			_, err = rdb.JSONArrInsert(vars["key"], path, index, data)
		}

		if err != nil {
			apiError("Failed to insert record", err, w)
			return
		}
		log.Println("Success")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}
	return f
}
