package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/fsnotify/fsnotify"

	"github.com/spf13/viper"
)

var (
	//possible valid security values
	securityValues = []string{"aes", "stupid", "none", ""}
)

// setup some default values for viper
func setViperDefaults() {

	//whether to output normal traffic logs
	viper.SetDefault("log", true)

	//port to run the leaderboard on
	viper.SetDefault("port", 4000)

	//enable the autocert bot
	viper.SetDefault("https", false)
	viper.RegisterAlias("production", "https")

	//domain name to be used for autocert
	//viper.SetDefault("domain", "www.mysecurewebsite.com)

	//location to store the autocerts cache, if needed
	viper.SetDefault("autocert_location", ".")

	//name for your game, shown in webview
	//viper.SetDefault("game_name", "")

	//make the HTML webview the default output on a GET request
	viper.SetDefault("webview", true)

	//limit long names. set to 0 for unlimited
	viper.SetDefault("max_name_length", 20)

	//location where csv file with score records is stored
	viper.SetDefault("csv_name", "scores.csv")

	//level of encryption to expect from incoming requests
	// "aes", "none", ""
	viper.SetDefault("security", "aes")

	//key to use when using aes encryption
	//viper.SetDefault("aes_key", "")

	//value to ensure correct aes key was used on received data
	viper.SetDefault("aes_checksum", "leaderboard")

	//use either json input or POST form input
	// "json", "form"
	viper.SetDefault("input_type", "json")

	// subroute to serve requests on, if any
	viper.SetDefault("leaderboard_path", "/")

	//ensure aes key length requirements
	err := ensureAESKeyLength(viper.GetString("aes_key"))
	if err != nil {
		panic(err)
	}

	//ensure security is a valid value
	sec := viper.GetString("security")
	validSec := false
	for _, v := range securityValues {
		if v == sec {
			validSec = true
			break
		}
	}
	if !validSec {
		panic(fmt.Sprintf("invalid value '%s' for 'security', must be one of [%v]", sec, securityValues))
	}
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
	setViperDefaults()

}
