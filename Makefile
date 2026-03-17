build:
	go build -o bin/aegiscourt ./cmd/aegiscourt

test:
	go test ./... -v

lint:
	golangci-lint run

clean:
	rm -rf bin/

docker-build:
	docker build -t aegiscourt .

build-linux:
	GOOS=linux go build -o bin/aegiscourt ./cmd/aegiscourt

cross-build:
	GOOS=linux GOARCH=amd64 go build -o bin/aegiscourt-linux-amd64 ./cmd/aegiscourt
	GOOS=darwin GOARCH=arm64 go build -o bin/aegiscourt-darwin-arm64 ./cmd/aegiscourt
	GOOS=windows GOARCH=amd64 go build -o bin/aegiscourt-windows-amd64.exe ./cmd/aegiscourt