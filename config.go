package main

import "github.com/spf13/viper"

// setup some default values for viper
func setViperConfig() {
	viper.SetConfigName("leaderboard")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	//whether to output normal traffic logs
	viper.SetDefault("log", true)

	//port to run the leaderboard on
	viper.SetDefault("port", 4000)

	//enable the autocert bot
	viper.SetDefault("https", false)
	viper.RegisterAlias("production", "https")

	//location to store the autocerts cache, if needed
	viper.SetDefault("autocert_location", ".")

	//name for your game, shown in webview
	//viper.SetDefault("game_name", "")

	//make the HTML webview the default output on a GET request
	viper.SetDefault("web_view", true)

	//filter profane names
	viper.SetDefault("name_filter", false)

	//location where csv file with scoer records is stored
	viper.SetDefault("csv_name", "scores.csv")

	//level of encryption to expect from incoming requests
	viper.SetDefault("security", "aes")

	//key to use when using aes encryption
	//viper.SetDefault("aes_key", "")
}
