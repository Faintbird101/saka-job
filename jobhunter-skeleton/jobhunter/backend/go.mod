module github.com/yourname/jobhunter/backend

go 1.25.0

// deps to add once you start:
//   github.com/go-chi/chi/v5   (router)
//   github.com/jackc/pgx/v5    (postgres)
//   github.com/joho/godotenv   (local .env loading)

require (
	github.com/go-chi/chi/v5 v5.3.1
	github.com/jackc/pgx/v5 v5.10.0
	github.com/joho/godotenv v1.5.1
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.29.0 // indirect
)
