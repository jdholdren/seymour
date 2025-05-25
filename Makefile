.PHONY: start yac

up:
	docker compose up

build:
	docker compose build

build-and-restart:
	docker compose build && docker compose restart
