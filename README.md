# Score Creeper
A web-based leaderboard server for any game.

# Security
Score Creeper optionally allows for securing messages with an AES key, ensuring that only players actually running your game can 
send requests to the server. You must enable this option in the config.


# Configuring
Score Creeper is configured via a `*.json` file that exists in the same directory as the executable. By default, this file should be named `leaderboard.json`. An example is
provided in `leaderboard_example.json`. Here are the configurable keys:
- `log` (default: true) - boolean, enables or disables traffic logging.
- `port` (default: 4000) - integer, network port to serve requests on.
- `https` (default: false) - boolean, enables certbot support for HTTPS. If this is true, must also configure `domain`.
- `domain` (default: "") - string, domain named on which this server is running, only used for certbot when `https` is true.
- `autocert_location` (default: ".") - string, filepath where to store autocert cache.
- `game_name` (default: "") - string, name of game that is shown in webview.
- `webview` (default: false) - boolean, enables webview response by default on any GET requests.
- `max_name_length` (default: 0) - integer, truncates any names longer than this value. Set to 0 for unlimited.
- `csv_name` (default: "scores.csv") - string, file name where scores are read to/written from. This will be created if it does not exist.
- `security` (default: "none") - string, must be one of ["aes", "none"]. Configure teh security type to use on incoming requests. If set to "aes", then "aes_key" and "aes_checksum" must also be configured. 
- `aes_key` (default: "") - string, this is the encryption key use to decrypt the value of incoming requests. You must encrypt the values of your request from within your game engine using this key before sending a score POST request. This key must match the one you use to encode your requests. This key must be 16, 24, or 32 characters exactly.
- `aes_checksum` (default: "leaderboard") - string, this checksum value is used to ensure your requests have been decoded properly. This is a value that is checked after decrypting the request, and is a constant, never changing value that is encrypted and passed along with any incoming POST request.  
- `input_type` (default: "json") - string, this value lets you accept requests bodies as json or form-data. Acceptable values are ["json", "form".]
- `leaderboard_path` (default: "/") - string, this is the path that this server will listen for requests on.

# Endpoints
All of these endpoints will assume an address of `localhost:4000` and a `leaderboard_path` setting of "/". All shown Request bodies will represent JSON request types, but the keys are the same as the field names when submitting form-data.
## Get all scores
This path does not need any encryption.

GET `/`

#### Query parameters
- `webview`: boolean, forces an html rendering of the scores.

#### Request body 

`empty`

#### Response Body
```
[
     {
         "place": 1,
         "name": "Katamari",
         "score": 555553
     },
     {
         "place": 2,
         "name": "NickCage",
         "score": 1234
     },
     ...
]
```


## Submit a new Score
If `security` = `aes`, all values in the request must be encrypted using your AES key defined in the `aes_key` config setting.

POST `/`

#### Request Body
*Note that the values for the request body are all strings, even the score.*

Unencrypted
```
{
	"score": "54354",
	"name": "Test",
	"checksum": "x"
}
```

Encrypted using `aes_key` = `DEADBEEFDEADBEEF`
```
{
	"score": "W9sC3KJAh1EwJaZFuDL1Ug==",
	"name": "SFyqbywa2D5FLi4diNEvyw==",
	"checksum": "Q/x1Mnrl+tU+61To4S77Hg=="
}
```


#### Response Body
```
{
    "allScores": [
        {
             "place": 1,
             "name": "Katamari",
             "score": 555553
        },
        {
            "place": 2,
            "name": "NickCage",
            "score": 1234
        },
        ...
    ],
    "rank": 8   //ranking of posted score
}
```

#Using with popular game engines
##Unity
These examples use the async/await features found in .NET 4.x. To enable these features, update your project settings by 
going to *Edit -> Project Settings -> Player -> Other Settings -> Api Compatibility Level* and setting it to **.NET 4.x**. 
###Getting all scores
Coming soon :)