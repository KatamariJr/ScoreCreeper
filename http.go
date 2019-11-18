package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

type playerValues struct {
	Score    string `json:"score"`
	Name     string `json:"name"`
	Checksum string `json:"checksum"`
}

type contextKey string

const (
	correlationIDContextKey contextKey = "correlationID"
)

func scorePostHandler(w http.ResponseWriter, r *http.Request) {
	uuid := uuid.New().String()

	var input playerValues

	switch viper.GetString("input_type") {
	case "json":
		json.NewDecoder(r.Body).Decode(&input)
	case "form":
		input.Score = r.FormValue("score")
		input.Name = r.FormValue("name")
		input.Checksum = r.FormValue("x")
	}

	score := input.Score
	name := input.Name
	checksum := input.Checksum

	logMessage(r.Context(), fmt.Sprintf("Incoming score=%s name=%s checksum=%s", score, name, checksum))

	maxLength := viper.GetInt("max_name_length")
	if maxLength > 0 {
		if len(name) > maxLength {
			name = name[:maxLength]
			logMessage(r.Context(), fmt.Sprintf("name truncated\n"))
		}
	}

	switch viper.GetString("security") {
	case "none", "":
		//no security
	case "stupid":
		//hacky checksum check: do not use this security measure. only here for backwards compatability
		err := validateDumbChecksum(score, name, checksum)
		if err != nil {
			http.Error(w, "Nope", http.StatusTeapot)
			logMessage(r.Context(), fmt.Sprintf("stupid checksum invalid:%s\n", err.Error()))
			return
		}
	case "aes":
		//validate using aes encryption
		var err error
		score, name, checksum, err = decryptValues([]byte(score), []byte(name), []byte(checksum))
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			logMessage(r.Context(), fmt.Sprintf("couldn't decrypt aes:%s\n", err.Error()))
			return
		}

		if checksum != viper.GetString("aes_checksum") {
			http.Error(w, "Nope", http.StatusTeapot)
			logMessage(r.Context(), fmt.Sprintf("aes checksum invalid:%s\n", err.Error()))
			return
		}
	default:
		//invalid security value set
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		logMessage(r.Context(), fmt.Sprintf("invalid value for 'security': %s\n", viper.GetString("security")))
		return
	}

	points, err := strconv.Atoi(score)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		logMessage(r.Context(), fmt.Sprintf("error converting to string: %v\n", err))
		return
	}

	err = logScore(name, points, uuid)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		logMessage(r.Context(), fmt.Sprintf("error logging score: %v\n", err))
		return
	}

	myResults, err := showScores(uuid)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		logMessage(r.Context(), fmt.Sprintf("error gathering scores: %v\n", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(myResults)
}

// loggerMiddleware will log information about teh current request as it comes in and as it finishes.
func loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		corrID := uuid.New()

		r = r.WithContext(context.WithValue(r.Context(), correlationIDContextKey, corrID))
		logMessage(r.Context(), fmt.Sprintf("Handling Request path=%s method=%s useragent=%s", r.URL.Path, r.Method, r.UserAgent()))
		next.ServeHTTP(w, r)
		logMessage(r.Context(), fmt.Sprintf("Request Complete duration=%d", time.Since(start)))
	})
}

// logMessage will automatically handle the correlation id and message formating.
func logMessage(ctx context.Context, message string) {
	corrID := ctx.Value(correlationIDContextKey)
	log.Printf("correlationID=%s msg=%s", corrID, message)
}

// route the incoming call to a json view or webview based on query param.
func getRouter(w http.ResponseWriter, r *http.Request) {
	if shouldShowWebView(r) {
		printScoreTable(w, r)
	} else {
		fetchAll(w, r)
	}
}

// return whatever the query params is set to or the value set in config
func shouldShowWebView(r *http.Request) bool {
	qp := r.URL.Query()
	if _, ok := qp["webview"]; ok {
		val, _ := strconv.ParseBool(qp.Get("webview"))
		return val
	}

	return viper.GetBool("webview")
}

// show an html view of the scores
func printScoreTable(w http.ResponseWriter, r *http.Request) {
	ranked := getAllRankedScoresFromTree()
	ret := "<table><th>Place</th><th>Name</th><th>Score</th>"
	for _, score := range ranked {
		ret += fmt.Sprintf("<tr><td>%v</td><td>%v</td><td>%v</td></tr>", score.Place, html.EscapeString(score.Name), score.Score)
	}
	ret += "</table>"

	w.Write([]byte(ret))
}

func fetchAll(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("fetching all scores")
	ranked := getAllRankedScoresFromTree()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ranked)
}
