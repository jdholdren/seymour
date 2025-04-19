.PHONY: start yac

# Runs the server binary
start:
	go run .

start-temporal:
	temporal server start-dev --db-filename temporal.db

yac:
	httpyac ./yac/local.http -i
