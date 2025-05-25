.PHONY: start yac

up:
	docker compose up

build:
	docker compose build

rb-citadel:
	docker compose build citadel && docker compose up -d --force-recreate citadel
