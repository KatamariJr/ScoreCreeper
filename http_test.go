package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"leaderboard/config"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/spf13/viper"
)

func TestMain(m *testing.M) {
	viper.Set(config.AESKey, "DEADBEEFDEADBEEF")
	viper.Set(config.CsvName, fmt.Sprintf(".%ctestdata%ctestscores.csv", os.PathSeparator, os.PathSeparator))
	config.SetViperDefaults()
	loadScoreTree()
	code := m.Run()
	os.Exit(code)
}

func buildAndSendRequest(h http.HandlerFunc, method, path string, body interface{}) (*http.Response, error) {
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(bodyJSON))

	return sendRequest(h, req)
}

func sendRequest(h http.HandlerFunc, r *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	h(w, r)
	return w.Result(), nil
}

//return an error if you didnt get a 200
func checkResponse(r *http.Response) error {
	if r.StatusCode != 200 {
		return fmt.Errorf("status: %s  code: %d", r.Status, r.StatusCode)
	}
	return nil
}

func TestHandler_NoSecurity(t *testing.T) {
	viper.Set(config.SecurityType, "none")
	viper.Set(config.InputType, "json")
	input := playerValues{
		Score: "123",
		Name:  "bob",
	}
	res, err := buildAndSendRequest(scorePostHandler, "POST", "/", input)
	if err != nil {
		t.Error(err)
	}

	err = checkResponse(res)
	if err != nil {
		t.Error(err)
	}
}
