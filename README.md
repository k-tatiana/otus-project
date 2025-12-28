# otus-project
Демо дипломного проекта: otus Highload Architect

## Docker Compose (Postgres, server, migrations) ✅

This repo includes a `docker-compose.yml` with three services:

- `postgres`: a Postgres database instance (postgres:15-alpine) with a persistent volume
- `server`: the Go server, built from the local `Dockerfile`
- `migrate`: uses the official `migrate` CLI image to apply SQL migrations found in the local `migrations/` directory

Quick start:

1. Create a directory for migrations (if needed):

```bash
mkdir -p migrations
```

2. Build and start the database (background):

```bash
docker compose up -d postgres
```

3. Apply migrations with the migrator (runs once and exits):

```bash
docker compose run --rm migrate
```

4. Build and run the server:

```bash
docker compose up --build -d server
```

5. View logs:

```bash
docker compose logs -f server
```

Notes:
- The server expects `DATABASE_URL` to be set as `postgres://otus:otuspass@postgres:5432/otusdb?sslmode=disable` inside the container. Adjust credentials or environment values (`POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`) in `docker-compose.yml` as needed.
- Add SQL migrations to `migrations/` using `golang-migrate` naming conventions; for example:
	- `1_init.up.sql` / `1_init.down.sql`

- A small sample migration has been included in `migrations/1_init.up.sql` and `migrations/1_init.down.sql` for convenience.

- Note on the `Dockerfile` build: the current `Dockerfile` attempts to build `./cmd/otus-project`. If your source `main.go` is at the repository root or the project layout differs, update `Dockerfile` accordingly (or create `cmd/otus-project/main.go`) so the binary builds successfully.

