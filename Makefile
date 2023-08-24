APP_NAME = car-park-bot

.PHONY: build run clean compose-build compose-run compose-clean

build:
	docker build -t $(APP_NAME) .

run:
	docker run -d --name $(APP_NAME) -p 7070:7070 $(APP_NAME)

clean:
	docker stop $(APP_NAME) $(DB_NAME)
	docker rm $(APP_NAME) $(DB_NAME)

compose-build:
	docker-compose build

compose-run:
	docker-compose up -d

compose-clean:
	docker-compose down