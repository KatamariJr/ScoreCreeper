package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/FX-HAO/GoOST/ost"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"golang.org/x/crypto/acme/autocert"
)

var (
	scoresLock sync.RWMutex
	scoreTree  *ost.OST
	fileLock   sync.RWMutex
)

const webdomain = "www.mysecurewebsite.com"

type RankedResult struct {
	uuid  string
	Place int    `json:"place"`
	Name  string `json:"name"`
	Score int    `json:"score"`
}

type UnrankedResult struct {
	ost.Item
	RowNum int
	Name   string
	Score  int
	UUID   string
}

// Less compares u to v
func (u UnrankedResult) Less(v ost.Item) bool {
	return u.Score < v.(UnrankedResult).Score
}

// Greater compares u to v
func (u UnrankedResult) Greater(v ost.Item) bool {
	return u.Score > v.(UnrankedResult).Score
}

// Equal compares u to v
func (u UnrankedResult) Equal(v ost.Item) bool {
	return u.Score == v.(UnrankedResult).Score
}

// Key returns the key
func (u UnrankedResult) Key() int {
	return u.RowNum
}

const (
	aesSecurity    = "aes"
	stupidSecurity = "stupid"
	noSecurity     = "none"
)

func main() {
	setViperConfig()

	router := mux.NewRouter() //.StrictSlash(true)
	router.HandleFunc("/", handler).Methods("POST")
	router.HandleFunc("/", getRouter).Methods("GET")

	fmt.Println("listening")

	//begin loading the score tree
	go func() {
		err := loadScoreTree()
		if err != nil {
			panic(err)
		}
	}()

	// add your listeners via http.Handle("/path", handlerObject)
	if viper.IsSet("https") && viper.GetBool("https") {
		log.Fatal(http.Serve(autocert.NewListener(webdomain), handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(router)))
		//log.Fatal(server.ListenAndServeTLS("", ""))
	} else {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", viper.GetInt("port")), router))
	}
}

// read the csv score data into the tree
func loadScoreTree() error {
	scoresLock.Lock()
	scores, err := readScores()
	defer scoresLock.Unlock()
	if err != nil {
		panic(err)
	}

	scoreTree = ost.New()
	for _, s := range scores {
		scoreTree.Insert(s)
	}

	fmt.Println("sorted set done")
	return nil
}

func logScore(name string, score int, uuid string) error {
	var f *os.File
	var err error

	scoresLock.Lock()
	defer scoresLock.Unlock()
	f, err = os.OpenFile(viper.GetString("csv_name"), os.O_RDWR|os.O_APPEND, 0660)
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

//read the scores from the score csv file and put them in memory
func readScores() ([]UnrankedResult, error) {
	fileLock.RLock()
	defer fileLock.RUnlock()
	fmt.Println("gonna read")
	f, err := os.Open(viper.GetString("csv_name"))
	if err != nil {
		if err == os.ErrNotExist {
			f, err = os.Create(viper.GetString("csv_name"))
			if err != nil {
				return nil, err
			}
		}
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
	return results, nil
}

func getAllRankedScoresFromTree() []RankedResult {
	//start := time.Now()

	ranked := []RankedResult{}

	i := 1
	iter := func(it ost.Item) bool {
		rr := it.(UnrankedResult)
		r := RankedResult{
			Place: i,
			Name:  rr.Name,
			Score: rr.Score,
			uuid:  rr.UUID,
		}
		ranked = append(ranked, r)
		i++
		return true
	}
	scoreTree.Descend(UnrankedResult{Score: 999999999}, UnrankedResult{Score: -9999999999}, iter)
	//delta := time.Since(start)

	//fmt.Printf("duration: %v\n", delta)
	return ranked
}

// get the top scores and a set number of scores surrounding the specified uuid
func showScores(uuid string) ([]RankedResult, error) {
	ranked := getAllRankedScoresFromTree()
	var myRank int
	for i, v := range ranked {
		if uuid == v.uuid {
			myRank = i
			break
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
}
