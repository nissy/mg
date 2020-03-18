NAME = mg
VERSION ?= v1.1.1
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
	sleep 10
	go test ./...

database-up: postgres-up mysql-up
database-down: postgres-down mysql-down

postgres-up:
	docker run -d \
		-p 5432:5432 \
		-e POSTGRES_DB=dbname \
		-e POSTGRES_USER=user \
		-e POSTGRES_PASSWORD=password \
		--name=$(NAME)-postgres postgres:9.6

postgres-down:
	docker rm -f $(NAME)-postgres

mysql-up:
	docker run -d \
	 	-p 3306:3306 \
		-e MYSQL_DATABASE=dbname \
		-e MYSQL_USER=user \
		-e MYSQL_PASSWORD=password \
		-e MYSQL_ROOT_PASSWORD=password \
		--name=$(NAME)-mysql mysql:5.7

mysql-down:
	docker rm -f $(NAME)-mysql
