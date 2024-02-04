.PHONY: build transfer

build:
	GOOS=linux GOARCH=amd64 go build -o downloader *.go

transfer: build
	scp downloader deck@100.105.200.56:"~/.bin"