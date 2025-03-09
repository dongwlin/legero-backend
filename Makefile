.PHONY: entgen build

entgen:
	go run -mod=mod entgo.io/ent/cmd/ent generate ./internal/ent/schema --idtype uint64

wiregen:
	wire gen ./internal/wire

build:
	go build -o ./bin/app .