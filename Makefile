# SHELL=/bin/bash

all: compile

clean:
	rm -f $(GOPATH)/bin/syringed
	rm -f $(GOPATH)/bin/syrctl

compile:
	@echo "Generating protobuf code..."

	@rm -rf api/exp/generated/ && mkdir -p api/exp/generated/

	@protoc -I/usr/local/include -I. \
	-I api/exp/definitions/ \
	-I$$GOPATH/src \
	-I$$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
	--grpc-gateway_out=logtostderr=true,allow_delete_body=true:. \
	api/exp/definitions/*.proto

	@mv api/exp/definitions/*.pb.gw.go api/exp/generated/

	@protoc -I api/exp/definitions/ -I. \
	-I api/exp/definitions/ \
	  api/exp/definitions/*.proto \
		-I$$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
	--go_out=plugins=grpc:api/exp/generated/

	@protoc -I/usr/local/include -I. \
	-I api/exp/definitions/ \
	  -I$$GOPATH/src \
	  -I$$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
	  --swagger_out=logtostderr=true,allow_delete_body=true:. \
	  api/exp/definitions/*.proto

	@echo "Generating swagger definitions..."
	@go generate ./api/exp/swagger/
	@hack/build-ui.sh

	@echo "Compiling syringe binaries..."
	@go install ./cmd/...

docker:
	docker build -t antidotelabs/syringe .
	docker push antidotelabs/syringe

test: 
	@go test ./... -cover

update:
	# Run the below to clear everything out and start from scratch
	# rm -rf ~/.glide && rm -rf vendor/ && rm -f glide.lock

	glide up -v

gengo:
	# You should only need to run this if the CRD API definitions change. Make sure you re-commit the changes once done.
	# https://blog.openshift.com/kubernetes-deep-dive-code-generation-customresources/

	rm -rf pkg/client/clientset && rm -rf pkg/client/informers && rm -rf pkg/client/listers
	
	vendor/k8s.io/code-generator/generate-groups.sh all \
	github.com/nre-learning/syringe/pkg/client \
	github.com/nre-learning/syringe/pkg/apis \
	k8s.cni.cncf.io:v1
