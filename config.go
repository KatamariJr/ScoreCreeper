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

	//ensure aes key length requirements
	err := ensureAESKeyLength(viper.GetString("aes_key"))
	if err != nil {
		panic(err)
	}
}
