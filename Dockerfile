# FROM golang:alpine
# WORKDIR /app


# ADD . /app
# RUN mkdir -p /usr/local/include
# RUN apk add make bash protobuf git curl
# RUN ls -lha /protobuf
# RUN go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
# RUN go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
# RUN go get -u github.com/golang/protobuf/protoc-gen-go
# RUN which protoc
# RUN ls -lha /go/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis/google/api
# RUN cd /app && go install ./cmd/...
# ENTRYPOINT ./goapp


FROM golang:1.10 as build-env
MAINTAINER Matt Oswalt <matt@keepingitclassless.net> (@mierdin)

LABEL version="0.1"

env PATH $GOPATH/bin:$PATH

RUN apt-get update \
 && apt-get install -y git curl unzip

RUN curl -OL https://github.com/google/protobuf/releases/download/v3.2.0/protoc-3.2.0-linux-x86_64.zip && unzip protoc-3.2.0-linux-x86_64.zip -d protoc3 && chmod +x protoc3/bin/* && mv protoc3/bin/* /usr/local/bin && mv protoc3/include/* /usr/local/include/

RUN go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
RUN go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
RUN go get -u github.com/golang/protobuf/protoc-gen-go

# RUN echo $GOPATH/bin
# RUN ls -lhs $GOPATH/bin

# Install syringe
COPY . $GOPATH/src/github.com/nre-learning/syringe
RUN cd $GOPATH/src/github.com/nre-learning/syringe && make

RUN ls -lha /go/bin/

ENTRYPOINT ["/go/bin/syringed"]

# WIP multi-stage build for slimmer image

# FROM alpine:3.8
# WORKDIR /app
# RUN mkdir -p /app
# COPY --from=build-env /go/bin/syringed /usr/bin
# RUN ls -lha /usr/bin/syringed


# RUN echo $PATH
# RUN apk add --update bash

# RUN touch /entrypoint.sh
# RUN echo "/app/syringed" >> /entrypoint.sh

# RUN chmod +x /usr/bin/syringed
# RUN chmod +x /entrypoint.sh
# ENTRYPOINT ["syringed"]
