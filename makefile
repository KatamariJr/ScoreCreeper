run: 
	go build -o scorecreeper
	./scorecreeper

binaries:
	GOOS=linux \
	GOARCH=amd64 \
	go build -o scorecreeper
	GOOS=darwin \
	GOARCH=amd64 \
	go build -o scorecreeperMac.o
	GOOS=windows \
	GOARCH=amd64 \
	go build -o scorecreeperWin.exe