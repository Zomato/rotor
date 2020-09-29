


up_local:
	docker-compose -f ./deployment/local/docker-compose.yml up

down_local:
	docker-compose -f ./deployment/local/docker-compose.yml down

build_local:
	docker-compose -f ./deployment/local/docker-compose.yml build
