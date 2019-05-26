test:
	cd aws && go vet ./... && go test -v ./... -race -cover
