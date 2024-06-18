DB_URL=postgresql://postgres:postgres@localhost:5432/ffdb?sslmode=disable
PROTO_DIR = infrastructure/proto
PB_DIR = infrastructure/pb
PROTO_FILES = $(PROTO_DIR)/*.proto

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

proto: clean_pb create_pb_dir
	protoc --proto_path=$(PROTO_DIR) --go_out=$(PB_DIR) --go_opt=paths=source_relative \
	--go-grpc_out=$(PB_DIR) --go-grpc_opt=paths=source_relative \
	$(PROTO_FILES)

clean_pb:
	rm -rf $(PB_DIR)

create_pb_dir:
	mkdir -p $(PB_DIR)

evans:
	evans --host localhost --port 9090 -r repl

.PHONY: createdb dropdb migrateup migratedown migrateup1 migratedown1 new_migration sqlc test proto evans
