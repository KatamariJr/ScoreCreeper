package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

func handler(w http.ResponseWriter, r *http.Request) {
	requestUUID := uuid.New().String()
	uuid := uuid.New().String()

	var input struct {
		Score    string `json:"score"`
		Name     string `json:"name"`
		Checksum string `json:"checksum"`
	}

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

	fmt.Printf("correlation=%s date=%s msg=Incoming score=%s name=%s checksum=%s\n", requestUUID, time.Now().String(), score, name, checksum)

	maxLength := viper.GetInt("max_name_length")
	if maxLength > 0 {
		if len(name) > maxLength {
			name = name[:maxLength]
			fmt.Printf("correlation=%s msg=name truncated\n", requestUUID)
			return
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
			fmt.Printf("correlation=%s msg=checksum invalid:%s\n", requestUUID, err.Error())
			return
		}
	case "aes":
		//TODO(agreen) aes security
	}

	points, err := strconv.Atoi(score)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		fmt.Printf("correlation=%s msg=error converting to string: %v\n", requestUUID, err)
		return
	}

	err = logScore(name, points, uuid)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		fmt.Printf("correlation=%s msg=error logging score: %v\n", requestUUID, err)
		return
	}

	myResults, err := showScores(uuid)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		fmt.Printf("correlation=%s msg=error gathering scores: %v\n", requestUUID, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(myResults)
	fmt.Println("end of handler")
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
