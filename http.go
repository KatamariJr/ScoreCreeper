package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"leaderboard/security"
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

// scorePostHandler will accept a request to post a score.
func scorePostHandler(w http.ResponseWriter, r *http.Request) {
	uuid := uuid.New().String()

	var input playerValues

	inputType := viper.GetString("input_type")
	switch inputType {
	case "json":
		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			logMessage(r.Context(), fmt.Sprintf("couldn't decode request: %s", err.Error()))
			return
		}
	case "form":
		input.Score = r.FormValue("score")
		input.Name = r.FormValue("name")
		input.Checksum = r.FormValue("x")
	}

	score := input.Score
	name := input.Name
	checksum := input.Checksum

	logMessage(r.Context(), fmt.Sprintf("Incoming score post request input_type=%s score=%s name=%s checksum=%s", inputType, score, name, checksum))

	maxLength := viper.GetInt("max_name_length")
	if maxLength > 0 {
		if len(name) > maxLength {
			name = name[:maxLength]
			logMessage(r.Context(), fmt.Sprintf("name truncated"))
		}
	}

	//validate request via security
	err := security.ValidateRequestParams(score, name, checksum)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		logMessage(r.Context(), fmt.Sprintf("failed to validate request params: %v", err.Error()))
		return
	}

	points, err := strconv.Atoi(score)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		logMessage(r.Context(), fmt.Sprintf("error converting to string: %v", err))
		return
	}

	err = logScore(name, points, uuid)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		logMessage(r.Context(), fmt.Sprintf("error logging score: %v", err))
		return
	}

	myResults, err := showScores(uuid)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		logMessage(r.Context(), fmt.Sprintf("error gathering scores: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(myResults)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		logMessage(r.Context(), fmt.Sprintf("couldn't encode response: %s", err.Error()))
		return
	}
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

// logMessage will automatically handle the correlation id and message formatting.
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

	_, err := w.Write([]byte(ret))
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		logMessage(r.Context(), fmt.Sprintf("couldn't write response: %s", err.Error()))
		return
	}
}

func fetchAll(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("fetching all scores")
	ranked := getAllRankedScoresFromTree()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(ranked)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		logMessage(r.Context(), fmt.Sprintf("couldn't encode response: %s", err.Error()))
		return
	}
}
