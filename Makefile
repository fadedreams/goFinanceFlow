
DB_URL=postgresql://postgres:postgres@localhost:5432/ffdb?sslmode=disable

network:
	docker network create ff-network

pg:
	docker run --name postgres --network ff-network -p 5432:5432 -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -d postgres:14-alpine

makedb:
	docker exec -it postgres createdb --username=postgres --owner=postgres ffdb

rundb:
	docker start postgres

dropdb:
	docker exec -it postgres dropdb ffdb

migrateup:
	migrate -path foundation/db/migration -database "$(DB_URL)" -verbose up

migrateup1:
	migrate -path foundation/db/migration -database "$(DB_URL)" -verbose up 1

migratedown:
	migrate -path foundation/db/migration -database "$(DB_URL)" -verbose down

migratedown1:
	migrate -path foundation/db/migration -database "$(DB_URL)" -verbose down 1

new_migration:
	migrate create -ext sql -dir foundation/db/migration -seq $(name)

sqlc:
	sqlc generate

test:
	go test ./...

.PHONY: createdb dropdb migrateup migratedown migrateup1 migratedown1 new_migration
