FROM golang:1.12 as build-env

# Install additional dependencies
RUN apt-get update \
 && apt-get install -y git curl unzip
RUN curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v3.11.4/protoc-3.11.4-linux-x86_64.zip && unzip protoc-3.11.4-linux-x86_64.zip -d protoc3 && chmod +x protoc3/bin/* && mv protoc3/bin/* /usr/local/bin && mv protoc3/include/* /usr/local/include/

# Copy Antidote code
COPY . $GOPATH/src/github.com/nre-learning/antidote-core

RUN cd $GOPATH/src/github.com/nre-learning/antidote-core && make install_bins

RUN go get github.com/jteeuwen/go-bindata/...

ENV PATH $GOPATH/bin:$PATH
EXPOSE 8086

# Compile binaries
RUN cd $GOPATH/src/github.com/nre-learning/antidote-core && make

# Copy binaries into new minimalist image
FROM scratch
COPY --from=build-env /go/bin/antictl /usr/bin/antictl
COPY --from=build-env /go/bin/antidoted /usr/bin/antidoted
COPY --from=build-env /go/bin/antidote /usr/bin/antidote
CMD ["/usr/bin/antidoted"]
