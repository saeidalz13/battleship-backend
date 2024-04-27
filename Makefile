

new_migration:
	migrate create -ext sql -dir db/migration -seq battleship

# to run, make migrate_down n=N where N is N down migrations (N is NOT the number in the migration filename rather the number of the migrations you want to apply)
migrate_down:
	migrate -path db/migration -database  -verbose down $(n)