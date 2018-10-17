version := $(shell git describe  --always --tags --long --abbrev=8)
buildtime := $(shell date -u +%Y%m%d.%H%M%S)

all: kafka-health

kafka-health:
	CGO_ENABLED=0 GOOS=linux go build -v -ldflags "-X main.version=$(version)-$(buildtime)"

test:
	go test ./...

clean:
	rm -f kafka-health