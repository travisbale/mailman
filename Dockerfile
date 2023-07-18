FROM golang:1.20-alpine

WORKDIR /app

COPY . .

RUN mkdir -p bin && go build -o bin/mailman cmd/mailman/main.go

CMD ["./bin/mailman"]
