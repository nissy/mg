NAME = mg
VERSION ?= v0.0.3
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

bin/$(NAME): build

build:
	CGO_ENABLED=0 go build -i -ldflags="-s -w -X main.version=$(VERSION)" -o bin/$(NAME) cmd/$(NAME)/*.go

dist: build
	mkdir -p dist
	tar cfz dist/$(NAME)-$(VERSION)_$(GOOS)_$(GOARCH).tar.gz -C bin/ $(NAME)

clean:
	rm -rf bin dist

test: database-up go-test database-down

go-test:
	sleep 5
	go test ./...

database-up: postgres-up
database-down: postgres-down

postgres-up:
	docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=test --name=$(NAME)-postgres postgres:9.6

postgres-down:
	docker rm -f $(NAME)-postgres
