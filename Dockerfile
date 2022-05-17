FROM golang:1.16-alpine3.13

WORKDIR /app

COPY . .

RUN go mod download
RUN mkdir -p bin && go build -o bin/mailman cmd/mailman.go

CMD ["./bin/mailman"]
