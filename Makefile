test:
	cd aws && go vet ./... && go test -v ./... -race -cover
	cd awsEnv && go vet ./... && go test -v ./... -race -cover
