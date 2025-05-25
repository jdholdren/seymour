.PHONY: start yac

up:
	docker compose up

build-and-restart:
	docker compose build && docker compose restart
