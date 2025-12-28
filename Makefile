PHONY: .start-base
start-base:
	docker compose -f dev/docker-compose.yml --profile base up -d

PHONY: .start-maximum
start-maximum:
	docker compose -f dev/docker-compose.yml --profile maximum up -d