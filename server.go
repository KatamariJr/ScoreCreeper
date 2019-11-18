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

func main() {
	setViperConfig()

	router := mux.NewRouter() //.StrictSlash(true)
	path := viper.GetString("leaderboard_path")
	router.HandleFunc(path, scorePostHandler).Methods("POST")
	router.HandleFunc(path, getRouter).Methods("GET")
	if viper.GetBool("log") {
		router.Use(loggerMiddleware)
	}

	fmt.Println("listening")

	//begin loading the score tree
	go func() {
		err := loadScoreTree()
		if err != nil {
			panic(err)
		}
	}()

	// add your listeners via http.Handle("/path", handlerObject)
	if viper.IsSet("https") && viper.GetBool("https") && viper.IsSet("domain") {
		log.Fatal(http.Serve(autocert.NewListener(viper.GetString("domain")), handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(router)))
	} else {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", viper.GetInt("port")), router))
	}
}

// read the csv score data into the tree
func loadScoreTree() error {
	fmt.Println("sorted set done")

	scores, err := readScores()
	if err != nil {
		panic(err)
	}

	scoresLock.Lock()
	defer scoresLock.Unlock()
	scoreTree = ost.New()
	for _, s := range scores {
		scoreTree.Insert(s)
	}

	return nil
}

func logScore(name string, score int, uuid string) error {
	var errCh = make(chan error, 1)

	//log the score in the csv
	go func() {
		fileLock.Lock()
		defer fileLock.Unlock()
		f, err := os.OpenFile(viper.GetString("csv_name"), os.O_RDWR|os.O_APPEND, 0660)
		if err != nil {
			errCh <- err
		}
		defer f.Close()

		scoreStr := strconv.Itoa(score)
		if err != nil {
			errCh <- err
		}

		//uuid, name, score, time
		record := []string{uuid, name, scoreStr, time.Now().String()}
		w := csv.NewWriter(f)
		err = w.Write(record)
		if err != nil {
			errCh <- err
		}
		w.Flush()

		err = w.Error()
		if err != nil {
			errCh <- err
		}
		close(errCh)
	}()

	//log the score in the tree
	scoresLock.Lock()
	defer scoresLock.Unlock()

	unrankedRes := UnrankedResult{
		Name:  name,
		Score: score,
		UUID:  uuid,
	}
	scoreTree.Insert(unrankedRes)

	//catch up with csv error
	err := <-errCh
	if err != nil {
		return err
	}

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
	scoresLock.RLock()
	defer scoresLock.RUnlock()

	scoreTree.Descend(UnrankedResult{Score: 999999999}, UnrankedResult{Score: -9999999999}, iter)

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

	//get top 5 and specified uuid and four surrounding scores
	if myRank <= 4 {
		returnedResults = ranked[:10]
	} else if myRank >= len(ranked)-2 {
		returnedResults = append(ranked[:5], ranked[myRank-2:]...)
	} else {
		returnedResults = append(ranked[:5], ranked[myRank-2:myRank+3]...)
	}

	return returnedResults, nil
}
