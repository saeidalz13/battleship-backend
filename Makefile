# DOCKER
create_network:
	docker network create battleship_network
	
run_container:
	docker run --network battleship_network --name battleship_psql -p 8765:5432 --env-file ".docker_env" -d postgres:16.2 
	docker exec battleship_psql psql -U battleteam -c "create database battleship;" 

exec_db:
	docker exec -it battleship_psql psql -U battleteam

# MIGRATIONS
new_migration:
	migrate create -ext sql -dir db/migration -seq battleship
	
# to run, make migrate_down n=N where N is N down migrations (N is NOT the number in the migration filename rather the number of the migrations you want to apply)
# For security reasons, DATABASE_URL should be replaced by the developers on local drive
migrate_down:
	migrate -path db/migration -database $(uri) -verbose down $(n)

.PHONY: test flylogs

# Test
test:
	go test -v ./test -count=1

# Fly
# Log
flogs_prod:
	fly logs -a battleship-go-ios

flogs_staging:
	fly logs -a battleship-go-ios-staging


depstage:
	fly deploy --app battleship-go-ios-staging --dockerfile Dockerfile.staging

flyconsole:
	fly console --app $(APP)
## With games with titles -> like 10 wins => captain

flypdb:
#   Connect to production db
	fly pg connect -a battleship-db

# Websocket
fly ws_staging:
	websocat wss://battleship-go-ios-staging.fly.dev/battleship