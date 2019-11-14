package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

func handler(w http.ResponseWriter, r *http.Request) {
	requestUUID := uuid.New().String()
	uuid := uuid.New().String()

	score := r.FormValue("score")
	name := r.FormValue("name")
	checksum := r.FormValue("x")

	fmt.Printf("correlation=%s date=%s msg=Incoming score=%s name=%s checksum=%s\n", requestUUID, time.Now().String(), score, name, checksum)

	err := validateChecksum(score, name, checksum)
	if err != nil {
		http.Error(w, "Nope", http.StatusTeapot)
		fmt.Printf("correlation=%s msg=checksum-invalid:%s\n", requestUUID, err.Error())
		return
	}

	fmt.Printf("Score: %s = Name: %s\n", score, name)
	points, err := strconv.Atoi(score)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		fmt.Printf("correlation=%s msg=error converting to string: %v\n", requestUUID, err)
		return
	}

	if len(name) >= 14 {
		name = name[:14]
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
	qp := r.URL.Query()
	if webview, _ := strconv.ParseBool(qp.Get("webview")); webview {
		printScoreTable(w, r)
	} else {
		fetchAll(w, r)
	}
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
