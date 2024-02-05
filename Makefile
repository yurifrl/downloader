.PHONY: build transfer

build:
	GOOS=linux GOARCH=amd64 go build -o dist/downloader cmd/*.go
	scp dist/downloader deck@100.105.200.56:"~/.bin"
	ssh deck@100.105.200.56 "chmod +x ~/.bin/downloader"
	scp config.yaml "deck@100.105.200.56":"~/Emulation"