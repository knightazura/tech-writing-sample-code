package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"
)

var storage map[int64]Transaction = map[int64]Transaction{
	1: {
		ID:     1,
		UserID: 1,
		Name:   "Buy iPhone 13",
		Items: []string{
			"iPhone 13",
			"Clear Case",
		},
		Amount:    10000000,
		CreatedAt: time.Now().Add(time.Hour * -1).Unix(),
	},
	2: {
		ID:     2,
		UserID: 0,
		Name:   "Anonymous buy T-Shirts",
		Items: []string{
			"T-Shirt Bugs Bunny White Colour",
			"T-Shirt Tweety Sun Colour",
		},
		Amount:    200000,
		CreatedAt: time.Now().Add(time.Minute * -30).Unix(),
	},
}

func main() {
	// setup JSON handler for logger and set level to DEBUG
	logJsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	// create structured logger instance
	logger := slog.New(logJsonHandler)
	// use the structured logger for default logger
	slog.SetDefault(logger)

	// create HTTP server that run in port 8000
	httpServer := &http.Server{
		Addr:        ":8000",
		ReadTimeout: time.Second * 30,
	}
	httpServer.Handler = reqLogMiddleware(getHandler())

	log.Println("server running on port 8000")
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("failed to start server, due: %v", err)
	}

}

func getHandler() http.Handler {
	router := http.NewServeMux()

	router.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		SuccessResponse(w, "Berry nice!")
	})
	router.HandleFunc("GET /transactions", func(w http.ResponseWriter, r *http.Request) {
		var list []Transaction
		for _, trx := range storage {
			list = append(list, trx)
		}

		SuccessResponse(w, list)
	})
	router.HandleFunc("POST /transactions", func(w http.ResponseWriter, r *http.Request) {
		// determine new transaction ID
		currTotalTrx := len(storage)
		newTrxID := currTotalTrx + 1

		// read payload
		var payload Transaction
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			// log the result with warn level
			slog.Error("fail decode request body", slog.Any("error", err))
			FailResponse(w, fmt.Errorf("fail decode request body"), http.StatusBadRequest)
			return
		}

		// transaction must have user ID if it's not anonymous transaction
		anonTrx, _ := strconv.ParseBool(r.URL.Query().Get("anonymous"))
		if !anonTrx && payload.UserID == 0 {
			// log the error
			slog.Error("transaction has empty user ID", slog.Any("payload", payload))
			FailResponse(w, fmt.Errorf("must provide user ID"), http.StatusBadRequest)
			return
		}

		// store transaction into storage
		newTrx := Transaction{
			ID:        int64(newTrxID),
			UserID:    payload.UserID,
			Name:      payload.Name,
			Items:     payload.Items,
			Amount:    payload.Amount,
			CreatedAt: time.Now().Unix(),
		}
		storage[int64(newTrxID)] = newTrx

		SuccessResponse(w, newTrx)
	})
	router.HandleFunc("GET /transactions/{id}", func(w http.ResponseWriter, r *http.Request) {
		// get transaction ID from path
		trxIdStr := r.PathValue("id")
		trxId, err := strconv.ParseInt(trxIdStr, 10, 64)
		if err != nil {
			// log the error
			slog.Error("invalid transaction id", slog.String("transaction_id", trxIdStr))
			FailResponse(w, fmt.Errorf("invalid transaction id"), http.StatusBadRequest)
			return
		}

		trx, ok := storage[trxId]
		if !ok {
			// log the result with warn level
			slog.Warn("transaction not found", slog.Int64("transaction_id", trxId))
			FailResponse(w, fmt.Errorf("transaction not found"), http.StatusNotFound)
			return
		}

		SuccessResponse(w, trx)
	})
	return router
}

func reqLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		if r.URL.Path != "/health" {
			slog.Info(
				fmt.Sprintf("request processing time: %v", time.Since(start)),
				slog.String("req_method", r.Method),
				slog.String("req_path", r.URL.Path),
				slog.String("req_query", r.URL.RawQuery),
			)
		}
	})
}

type Response struct {
	Ok           bool        `json:"ok"`
	Data         interface{} `json:"data,omitempty"`
	ErrorMessage string      `json:"error_message,omitempty"`
}

func FailResponse(w http.ResponseWriter, err error, errStatusCode int) {
	// build the response
	resp := Response{
		Ok:           false,
		ErrorMessage: err.Error(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(errStatusCode)
	json.NewEncoder(w).Encode(resp)
}

func SuccessResponse(w http.ResponseWriter, data interface{}) {
	// build the response
	resp := Response{
		Ok:   true,
		Data: data,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
