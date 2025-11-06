FROM golang:1.23 AS build

RUN apt-get update && apt-get install -y unzip

# Install protoc
RUN curl -LO https://github.com/protocolbuffers/protobuf/releases/download/v28.3/protoc-28.3-linux-x86_64.zip \
	&& unzip protoc-28.3-linux-x86_64.zip -d /usr/local && rm protoc-28.3-linux-x86_64.zip

# Install Go plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.35.2 \
	&& go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

ENV PATH="/go/bin:${PATH}"

WORKDIR /workspace
ENTRYPOINT ["protoc"]
