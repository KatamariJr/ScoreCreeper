run: 
	go build 
	HTTPS="false" \
	WEBVIEW="false" \
	SECURITY="none" \
	./leaderboard