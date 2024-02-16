.PHONY: clean hello darwin windows linux all

hello:
	echo "Hello Golang!!"
clean:
	rm -rf ./out
darwin:
	GOOS=darwin GOARCH=amd64 go build -o ./out/proctracker-darwin .
windows:
	GOOS=windows GOARCH=amd64 go build -o ./out/proctracker-win.exe .
linux:
	GOOS=linux GOARCH=amd64 go build -o ./out/proctracker-linux .
all: clean darwin windows linux
