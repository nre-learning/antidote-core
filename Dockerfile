FROM golang:1.12 as build-env
MAINTAINER Matt Oswalt <matt@keepingitclassless.net> (@mierdin)

# Install additional dependencies
RUN apt-get update \
 && apt-get install -y git curl unzip
RUN curl -OL https://github.com/google/protobuf/releases/download/v3.2.0/protoc-3.2.0-linux-x86_64.zip && unzip protoc-3.2.0-linux-x86_64.zip -d protoc3 && chmod +x protoc3/bin/* && mv protoc3/bin/* /usr/local/bin && mv protoc3/include/* /usr/local/include/

# Copy Syringe code
COPY . $GOPATH/src/github.com/nre-learning/syringe

# Compile binaries from vendored libs
# (This is important so that we keep the version of our tools and our vendored libraries identical)
RUN cd $GOPATH/src/github.com/nre-learning/syringe/vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway/ \
    && go install ./...
RUN cd $GOPATH/src/github.com/nre-learning/syringe/vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/ \
    && go install ./...

RUN go get -u github.com/golang/protobuf/protoc-gen-go
RUN go get -u github.com/envoyproxy/protoc-gen-validate
RUN go get github.com/jteeuwen/go-bindata/...

env PATH $GOPATH/bin:$PATH
EXPOSE 8086

# Install syringe
RUN cd $GOPATH/src/github.com/nre-learning/syringe && make

# Copy binaries into new minimalist image
# TODO(mierdin): DNS lookups not working right in scratch. I tried debian and it just blew chunks. Need to look into a solution for this
FROM scratch
COPY --from=build-env /go/bin/syringed /usr/bin/syringed
COPY --from=build-env /go/bin/syringed-mock /usr/bin/syringed-mock
COPY --from=build-env /go/bin/syrctl /usr/bin/syrctl
COPY --from=build-env /go/bin/antidote-stats /usr/bin/antidote-stats
CMD ["/usr/bin/syringed"]
