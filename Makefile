run:
	go run ./cmd/gofetch
build: 
	go build -o ./bin/gofetch.exe ./cmd/gofetch
test: 
	./bin/gofetch.exe "https://www.google.com" "https://pkg.go.dev/golang.org/x/net/html#pkg-examples" "https://zerodha.com"