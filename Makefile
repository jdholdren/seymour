.PHONY: start yac

up:
	docker compose up

build:
	docker compose build

rb-citadel:
	docker compose build citadel && \
	docker compose up -d --force-recreate citadel

rb-agg:
	docker compose build agg_server && \
	docker compose build agg_worker && \
	docker compose up -d --force-recreate agg_server && \
	docker compose up -d --force-recreate agg_worker
