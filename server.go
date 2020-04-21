package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"leaderboard/config"
	"leaderboard/security"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

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
	rowNum     int
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
	setupConfig()

	router := mux.NewRouter() //.StrictSlash(true)
	path := viper.GetString(config.LeaderboardPath)

	// add your listeners via http.Handle("/path", handlerObject)
	router.HandleFunc(path, scorePostHandler).Methods("POST")
	router.HandleFunc(path, getRouter).Methods("GET")

	if viper.GetBool(config.Log) {
		router.Use(loggerMiddleware)
	}

	//begin loading the score tree
	go func() {
		err := loadScoreTree()
		if err != nil {
			panic(err)
		}
	}()
	port := viper.GetInt(config.Port)
	log.Printf("listening on port '%d'", port)

	log.Printf("using security format '%s'", viper.GetString(config.SecurityType))
	log.Printf("using input type '%s'", viper.GetString(config.InputType))

	if viper.IsSet(config.HTTPS) && viper.GetBool(config.HTTPS) && viper.IsSet(config.Domain) {
		log.Fatal(http.Serve(autocert.NewListener(viper.GetString(config.Domain)), handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(router)))
	} else {

		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), router))
	}
}

// read the csv score data into the tree
func loadScoreTree() error {

	scores, err := readScores()
	if err != nil {
		panic(err)
	}

	rowNum = len(scores)

	scoresLock.Lock()
	defer scoresLock.Unlock()
	scoreTree = ost.New()
	for _, s := range scores {
		scoreTree.Insert(s)
	}

	log.Println("score tree loaded in memory")

	return nil
}

func logScore(name string, score int, uuid string) error {
	var errCh = make(chan error, 1)

	//log the score in the csv
	go func() {
		fileLock.Lock()
		defer fileLock.Unlock()
		f, err := os.OpenFile(viper.GetString(config.CsvName), os.O_RDWR|os.O_APPEND, 0660)
		if err != nil {
			errCh <- err
			return
		}
		defer f.Close()

		scoreStr := strconv.Itoa(score)

		//uuid, name, score, time
		record := []string{uuid, name, scoreStr, time.Now().String()}
		w := csv.NewWriter(f)
		err = w.Write(record)
		if err != nil {
			errCh <- err
			return
		}
		w.Flush()

		err = w.Error()
		if err != nil {
			errCh <- err
			return
		}
		close(errCh)
	}()

	//log the score in the tree
	scoresLock.Lock()
	defer scoresLock.Unlock()

	rowNum++
	unrankedRes := UnrankedResult{
		RowNum: rowNum,
		Name:   name,
		Score:  score,
		UUID:   uuid,
	}
	scoreTree.Insert(unrankedRes)

	//catch up with csv error
	err := <-errCh
	if err != nil {
		return err
	}

	return nil
}

//read the scores from the score csv file and return as a slice
func readScores() ([]UnrankedResult, error) {
	fileLock.RLock()
	defer fileLock.RUnlock()

	csvFileName := viper.GetString(config.CsvName)
	log.Printf("reading scores from file '%s'", csvFileName)

	f, err := os.Open(csvFileName)
	if err != nil {
		if err == os.ErrNotExist {
			f, err = os.Create(csvFileName)
			if err != nil {
				return nil, err
			}
		}
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)

	results := []UnrankedResult{}
	localRowNum := 1
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
			RowNum: localRowNum,
			UUID:   record[0],
			Name:   record[1],
			Score:  sc,
		}

		results = append(results, result)
		localRowNum++
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
func showScores(uuid string) (int, []RankedResult, error) {
	ranked := getAllRankedScoresFromTree()
	var myRank int
	for i, v := range ranked {
		if uuid == v.uuid {
			myRank = i
			break
		}
	}

	return myRank + 1, ranked, nil
}

func setupConfig() {
	const configName = "leaderboard"
	var configLocations = []string{
		".",
	}

	//set config name and locations
	viper.SetConfigName(configName)
	viper.SetConfigType("json")
	for _, l := range configLocations {
		viper.AddConfigPath(l)
	}

	//set environment variable settings
	viper.SetEnvPrefix("LEADERBOARD")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Printf("Config file changed: %s", e.Name)
	})

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			//log.Printf("no config file '%s' found, searched the following directories %v", configName, configLocations)
		} else {
			log.Fatal(fmt.Errorf("fatal error in config file: %w", err))
		}
	}

	//default values
	config.SetViperDefaults()

	//validate values
	security.ValidateSecurityType()

}
