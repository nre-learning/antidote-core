SHELL=/bin/bash

all: compile

clean:
	rm -f $(GOPATH)/bin/syringed
	rm -f $(GOPATH)/bin/syrctl

compile:
	rm -rf api/exp/generated/ && mkdir -p api/exp/generated/ && protoc -I api/exp/definitions/ api/exp/definitions/* --go_out=plugins=grpc:api/exp/generated/
	go install ./cmd/...

test: 
	go test ./... -cover

update:
	glide up -v

gengo:
	rm -rf pkg/client/clientset && \
	vendor/k8s.io/code-generator/generate-groups.sh all \
	github.com/nre-learning/syringe/pkg/client \
	github.com/nre-learning/syringe/pkg/apis \
	kubernetes.com:v1