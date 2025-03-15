.PHONY: start yac

# TODO: Add a target for binary

# Runs the server binary
start:
	go run .

yac:
	httpyac ./yac/local.http -i
