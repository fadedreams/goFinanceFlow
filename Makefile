
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
	migrate -path infrastructure/db/migration -database "$(DB_URL)" -verbose up

migrateup1:
	migrate -path infrastructure/db/migration -database "$(DB_URL)" -verbose up 1

migratedown:
	migrate -path infrastructure/db/migration -database "$(DB_URL)" -verbose down

migratedown1:
	migrate -path infrastructure/db/migration -database "$(DB_URL)" -verbose down 1

new_migration:
	migrate create -ext sql -dir infrastructure/db/migration -seq $(name)

sqlc:
	sqlc generate

test:
	go test ./...


proto:
	protoc --proto_path=infrastructure/proto --go_out=infrastructure/pb --go_opt=paths=source_relative \
	--go-grpc_out=infrastructure/pb --go-grpc_opt=paths=source_relative \
	infrastructure/proto/*.proto

.PHONY: createdb dropdb migrateup migratedown migrateup1 migratedown1 new_migration sqlc test proto
