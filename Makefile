# SHELL=/bin/bash

all: compile

clean:
	rm -f $(GOPATH)/bin/syringed
	rm -f $(GOPATH)/bin/syrctl

compiledocker:
	# shopt -s extglob
	# echo rm foo.!(*.)

	rm -rf api/exp/generated/ && mkdir -p api/exp/generated/

	protoc -I/usr/local/include -I. \
		-I api/exp/definitions/ \
		-I$$GOPATH/src \
		-I$$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--grpc-gateway_out=logtostderr=true,allow_delete_body=true:. \
		api/exp/definitions/*.proto

	mv api/exp/definitions/*.pb.gw.go api/exp/generated/

	protoc -I api/exp/definitions/ -I. \
	  -I api/exp/definitions/ \
	  api/exp/definitions/*.proto \
		-I$$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
	  --go_out=plugins=grpc:api/exp/generated/

	protoc -I/usr/local/include -I. \
	  -I api/exp/definitions/ \
	  -I$$GOPATH/src \
	  -I$$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
	  --swagger_out=logtostderr=true,allow_delete_body=true:. \
	  api/exp/definitions/*.proto

	mv api/exp/definitions/*.swagger.json api/exp/generated/
	go install -ldflags "-linkmode external -extldflags -static" ./cmd/...

compile:
	# shopt -s extglob
	rm -rf api/exp/generated/ && mkdir -p api/exp/generated/

	# Generate go-client code for working with CRD
	# vendor/k8s.io/code-generator/generate-groups.sh all github.com/nre-learning/syringe/pkg/client github.com/nre-learning/syringe/pkg/apis kubernetes.com:v1

	protoc -I/usr/local/include -I. \
	-I api/exp/definitions/ \
	-I$$GOPATH/src \
	-I$$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
	--grpc-gateway_out=logtostderr=true,allow_delete_body=true:. \
	api/exp/definitions/*.proto

	mv api/exp/definitions/*.pb.gw.go api/exp/generated/

	protoc -I api/exp/definitions/ -I. \
	-I api/exp/definitions/ \
	  api/exp/definitions/*.proto \
		-I$$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
	--go_out=plugins=grpc:api/exp/generated/

	protoc -I/usr/local/include -I. \
	-I api/exp/definitions/ \
	  -I$$GOPATH/src \
	  -I$$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
	  --swagger_out=logtostderr=true,allow_delete_body=true:. \
	  api/exp/definitions/*.proto

	mv api/exp/definitions/*.swagger.json api/exp/generated/
	# go install -ldflags "-linkmode external -extldflags -static" ./cmd/...
	go install ./cmd/...

docker:
	docker build -t antidotelabs/syringe .
	docker push antidotelabs/syringe

test: 
	go test ./... -cover

update:
	# rm -rf ~/.glide && rm -rf vendor/ && rm -f glide.lock
	glide up -v

# You should only need to run this if the CRD API definitions change. Make sure you re-commit the changes once done.
gengo:

	# https://blog.openshift.com/kubernetes-deep-dive-code-generation-customresources/

	rm -rf pkg/client/clientset && rm -rf pkg/client/informers && rm -rf pkg/client/listers
	
	vendor/k8s.io/code-generator/generate-groups.sh all \
	github.com/nre-learning/syringe/pkg/client \
	github.com/nre-learning/syringe/pkg/apis \
	k8s.cni.cncf.io:v1