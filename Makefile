.PHONY: all clean dipatch didiff

all: deps dipatch didiff

deps:
	dep ensure

clean:
	rm -rf bin/

dipatch:
	go build -o bin/dipatch cmd/create-docker-patch/main.go

didiff:
	go build -o bin/didiff cmd/apply-docker-patch/main.go

