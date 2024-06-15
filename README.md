### establish db
- make network
- make pg
- make makedb
- make sqlc
- make migrateup

#### generate tests
- mockgen -package mockdb -destination mock/store.go  github.com/fadedreams/gofinanceflow/infrastructure/db/sqlc Store 
- mockgen -source=sqlc/store.go -destination=mock_db/mock_db.go -package=mock_db #nope

