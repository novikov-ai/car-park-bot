FROM golang:1.19

WORKDIR /car-park-bot

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o car-park-bot .

EXPOSE 7070

CMD ["./car-park-bot"]