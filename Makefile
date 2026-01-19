PHONY: .start-base
start-base:
	docker compose -f dev/docker-compose.yml --profile base build
	docker compose -f dev/docker-compose.yml --profile base up -d

PHONY: .start-maximum
start-maximum:
	docker compose -f dev/docker-compose.yml --profile maximum build
	docker compose -f dev/docker-compose.yml --profile maximum up -d

PHONY: .stop
stop:
	docker compose -f dev/docker-compose.yml down
