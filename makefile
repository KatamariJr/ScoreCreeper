run: 
	go build 
	HTTPS="false" \
	WEBVIEW="false" \
	SECURITY="none" \
	AES_KEY="DEADBEEFDEADBEEF" \
	LOG="true" \
	./leaderboard