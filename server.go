package main

import (
	"bytes"
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/acme/autocert"
)

var scoresLock sync.RWMutex

const webdomain = "www.mysecurewebsite.com"

type RankedResult struct {
	Place int    `json:"place"`
	Name  string `json:"name"`
	Score int    `json:"score"`
}

type UnrankedResult struct {
	Name  string
	Score int
	UUID  string
}

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

func main() {
	router := mux.NewRouter() //.StrictSlash(true)
	router.HandleFunc("/", handler).Methods("POST")
	router.HandleFunc("/", getRouter).Methods("GET")

	fmt.Println("listening")

	// add your listeners via http.Handle("/path", handlerObject)
	log.Fatal(http.Serve(autocert.NewListener(webdomain), handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(router)))
	//log.Fatal(server.ListenAndServeTLS("", ""))

	//log.Fatal(http.ListenAndServe(":4000", router))
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
	res, err := readScores()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		fmt.Printf("msg=error fetching all scores: %v\n", err)
		return
	}
	ranked := rankScores(res)
	ret := "<table><th>Place</th><th>Name</th><th>Score</th>"
	for _, score := range ranked {
		ret += fmt.Sprintf("<tr><td>%v</td><td>%v</td><td>%v</td></tr>", score.Place, html.EscapeString(score.Name), score.Score)
	}
	ret += "</table>"

	w.Write([]byte(ret))
}

func validateChecksum(score, name, checksum string) error {
	if len(checksum) != 33 {
		return errors.New("invalid checksum: wrong length")
	}

	extraChar := checksum[9]
	if extraChar != 'a' {
		return errors.New("invalid checksum: missing char")
	}

	incomingHash := checksum[:9] + checksum[10:]

	md5 := md5.Sum([]byte(name + score))
	realHash := hex.EncodeToString(md5[:])

	if !bytes.Equal([]byte(realHash), []byte(incomingHash)) {
		fmt.Println(incomingHash)
		fmt.Println(realHash)
		return errors.New("invalid checksum: no match")
	}

	return nil
}

func logScore(name string, score int, uuid string) error {
	var f *os.File
	var err error

	scoresLock.Lock()
	defer scoresLock.Unlock()
	f, err = os.OpenFile("scores.csv", os.O_RDWR|os.O_APPEND, 0660)
	if err != nil {
		return err
	}
	defer f.Close()

	scoreStr := strconv.Itoa(score)
	if err != nil {
		return err
	}

	//uuid, name, score, time
	record := []string{uuid, name, scoreStr, time.Now().String()}
	w := csv.NewWriter(f)
	err = w.Write(record)
	if err != nil {
		return err
	}
	w.Flush()

	err = w.Error()
	if err != nil {
		return err
	}
	fmt.Println("written")

	return nil
}

func readScores() ([]UnrankedResult, error) {
	scoresLock.RLock()
	f, err := os.Open("scores.csv")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)

	results := []UnrankedResult{}

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		sc, err := strconv.Atoi(record[2])
		if err != nil {
			return nil, err
		}

		result := UnrankedResult{
			UUID:  record[0],
			Name:  record[1],
			Score: sc,
		}

		results = append(results, result)
	}
	scoresLock.RUnlock()
	return results, nil
}

func fetchAll(w http.ResponseWriter, r *http.Request) {
	res, err := readScores()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		fmt.Printf("msg=error fetching all scores: %v\n", err)
		return
	}
	ranked := rankScores(res)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ranked)
}

func rankScores(scores []UnrankedResult) []RankedResult {
	//sort by score
	sort.Slice(scores, func(i, j int) bool { return scores[i].Score > scores[j].Score })

	ranked := make([]RankedResult, len(scores))
	for i, v := range scores {
		ranked[i] = RankedResult{
			Name:  v.Name,
			Place: i,
			Score: v.Score,
		}
	}

	return ranked
}

func showScores(uuid string) ([]RankedResult, error) {
	//get all values from csv
	scores, err := readScores()
	if err != nil {
		return nil, err
	}

	//sort by score
	sort.Slice(scores, func(i, j int) bool { return scores[i].Score > scores[j].Score })

	var myRank int
	//add rank amount
	ranked := make([]RankedResult, len(scores))
	for i, v := range scores {
		if uuid == v.UUID {
			myRank = i
		}
		ranked[i] = RankedResult{
			Name:  v.Name,
			Place: i,
			Score: v.Score,
		}
	}

	var returnedResults []RankedResult

	//get top 5 and specified uuid and four surorounding scores
	if myRank <= 4 {
		returnedResults = ranked[:10]
	} else if myRank >= len(ranked)-2 {
		returnedResults = append(ranked[:5], ranked[myRank-2:]...)
	} else {
		returnedResults = append(ranked[:5], ranked[myRank-2:myRank+3]...)
	}

	return returnedResults, nil
	//return ranked, nil
}
