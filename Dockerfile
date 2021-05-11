FROM golang:1.16-alpine3.13

WORKDIR /app

COPY . .

RUN go mod download
RUN mkdir bin && go build -o bin/mailman

CMD ["./bin/mailman"]
