# DOCKER
create_network:
	docker network create battleship_network
run_container:
	docker run --network battleship_network --name battleship_psql -p 8765:5432 --env-file ".docker_env" -d postgres:16.2 
create_db:
	docker exec battleship_psql psql -U battleteam -c "create database if not exists battleship;" 

# MIGRATIONS
new_migration:
	migrate create -ext sql -dir db/migration -seq battleship
	
# to run, make migrate_down n=N where N is N down migrations (N is NOT the number in the migration filename rather the number of the migrations you want to apply)
# For security reasons, DATABASE_URL should be replaced by the developers on local drive
migrate_down:
	migrate -path db/migration -database DATABASE_URL -verbose down $(n)

.PHONY: test flylogs

# Test
test:
	go test -v ./test -count=1

# Log
flylogs:
	fly logs -a battleship-go-ios

depstage:
	fly deploy --app battleship-go-ios-staging --dockerfile Dockerfile.staging


## With games with titles -> like 10 wins => captain