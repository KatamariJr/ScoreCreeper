package config

import (
	"github.com/spf13/viper"
)

//viper string values for all config settings
var (
	Log              = "log"               //output normal traffic logs
	Port             = "port"              // port to serve leaderboard requests on
	HTTPS            = "https"             //enable autocert bot
	Production       = "production"        // alias of HTTPS
	Domain           = "domain"            // domain to be used for autocert
	AutocertLocation = "autocert_location" // location to store autocert cache
	GameName         = "game_name"         // the name of your game, shown in webview
	WebviewEnabled   = "webview"           // show an html table on get request
	MaxNameLength    = "max_name_length"   //limit long names. set to 0 for unlimited
	CsvName          = "csv_name"          //location where csv file with score records is stored
	SecurityType     = "security"          //level of encryption to expect from incoming requests ["aes", "none", ""]
	AESKey           = "aes_key"           //key to use when using aes encryption
	AESChecksum      = "aes_checksum"      //value to ensure correct aes key was used on received data
	InputType        = "input_type"        //use either json input or POST form input ["json", "form"]
	LeaderboardPath  = "leaderboard_path"  // subroute to serve requests on, if any

)

// SetViperDefaults will setup some default values for viper
func SetViperDefaults() {
	viper.SetDefault(Log, true)
	viper.SetDefault(Port, 4000)
	viper.SetDefault(HTTPS, false)
	viper.RegisterAlias(Production, HTTPS)
	//viper.SetDefault(Domain, "www.mysecurewebsite.com)
	viper.SetDefault(AutocertLocation, ".")
	//viper.SetDefault(GameName, "")
	viper.SetDefault(WebviewEnabled, false)
	viper.SetDefault(MaxNameLength, 0)
	viper.SetDefault(CsvName, "scores.csv")
	viper.SetDefault(SecurityType, "none")
	//viper.SetDefault(AESKey, "")
	viper.SetDefault(AESChecksum, "leaderboard")
	viper.SetDefault(InputType, "json")
	viper.SetDefault(LeaderboardPath, "/")
}
