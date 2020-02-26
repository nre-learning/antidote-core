# SHELL=/bin/bash

TARGET_VERSION ?= latest

all: compile

compile:

	@echo "Generating protobuf code..."
	@rm -f pkg/ui/data/swagger/datafile.go
	@rm -f /tmp/datafile.go
	@rm -f cmd/antidote/buildinfo.go
	@rm -f cmd/antidoted/buildinfo.go
	@rm -f cmd/antictl/buildinfo.go
	@rm -rf api/exp/generated/ && mkdir -p api/exp/generated/
	@./compile-proto.sh

	@#https://stackoverflow.com/questions/34716238/golang-protobuf-remove-omitempty-tag-from-generated-json-tags/37335452#37335452
	@ls api/exp/generated/*.pb.go | xargs -n1 -IX bash -c 'sed s/,omitempty// X > X.tmp && mv X{.tmp,}'

	@echo "Generating swagger definitions..."
	@go generate ./api/exp/swagger/
	@hack/build-ui.sh

	@echo "Generating build info file..."
	@hack/gen-build-info.sh

	@echo "Compiling antidote binaries..."

	@# It doesn't seem like we need this. TODO(mierdin): Verify
	@#go install -ldflags "-linkmode external -extldflags -static" ./cmd/...

	@go install ./cmd/...

docker:
	docker build -t antidotelabs/antidote:$(TARGET_VERSION) .
	docker push antidotelabs/antidote:$(TARGET_VERSION)

test:
	@#This will run tests on all but the pkg package, if you want to limit this in the future.
	@#go test `go list ./... | grep -v pkg`

	@go test ./... -cover -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o=coverage.html

update:
	dep ensure

gengo:
	# You should only need to run this if the CRD API definitions change. Make sure you re-commit the changes once done.
	# https://blog.openshift.com/kubernetes-deep-dive-code-generation-customresources/

	rm -rf pkg/client/clientset && rm -rf pkg/client/informers && rm -rf pkg/client/listers
	
	vendor/k8s.io/code-generator/generate-groups.sh all \
	github.com/nre-learning/antidote-core/pkg/client \
	github.com/nre-learning/antidote-core/pkg/apis \
	k8s.cni.cncf.io:v1

	@# We need to play doctor on some of these files. Haven't figured out yet how to ensure hyphens are preserved in the
	@# fully qualified resource name, so we're just generating as normal, and then renaming files and replacing text
	@# as needed.

	@mv pkg/client/clientset/versioned/typed/k8s.cni.cncf.io/v1/networkattachmentdefinition.go \
		pkg/client/clientset/versioned/typed/k8s.cni.cncf.io/v1/network-attachment-definition.go

	@mv pkg/client/listers/k8s.cni.cncf.io/v1/networkattachmentdefinition.go \
		pkg/client/listers/k8s.cni.cncf.io/v1/network-attachment-definition.go

	@sed -i 's/networkattachmentdefinition/network-attachment-definition/g' \
		pkg/client/clientset/versioned/typed/k8s.cni.cncf.io/v1/network-attachment-definition.go \
		pkg/client/informers/externalversions/generic.go \
		pkg/client/listers/k8s.cni.cncf.io/v1/network-attachment-definition.go

	@sed -i 's/Resource: "networkattachmentdefinitions"/Resource: "network-attachment-definitions"/g' \
		pkg/client/clientset/versioned/typed/k8s.cni.cncf.io/v1/fake/fake_networkattachmentdefinition.go 



install_bins_linux:

	@curl -L https://github.com/grpc-ecosystem/grpc-gateway/releases/download/v1.5.1/protoc-gen-grpc-gateway-v1.5.1-linux-x86_64 -o $$GOPATH/bin/protoc-gen-grpc-gateway && chmod +x $$GOPATH/bin/protoc-gen-grpc-gateway
	@curl -L https://github.com/grpc-ecosystem/grpc-gateway/releases/download/v1.5.1/protoc-gen-swagger-v1.5.1-linux-x86_64 -o $$GOPATH/bin/protoc-gen-swagger && chmod +x $$GOPATH/bin/protoc-gen-swagger

install_bins_mac:

	@curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v3.2.0/protoc-3.2.0-osx-x86_64.zip && rm -rf protoc3 && unzip protoc-3.2.0-osx-x86_64.zip -d protoc3 && chmod +x protoc3/bin/* && sudo mv protoc3/bin/* /usr/local/bin && sudo mv protoc3/include/* /usr/local/include/
	@curl -L https://github.com/grpc-ecosystem/grpc-gateway/releases/download/v1.5.1/protoc-gen-grpc-gateway-v1.5.1-darwin-x86_64 -o $$GOPATH/bin/protoc-gen-grpc-gateway && chmod +x $$GOPATH/bin/protoc-gen-grpc-gateway
	@curl -L https://github.com/grpc-ecosystem/grpc-gateway/releases/download/v1.5.1/protoc-gen-swagger-v1.5.1-darwin-x86_64 -o $$GOPATH/bin/protoc-gen-swagger && chmod +x $$GOPATH/bin/protoc-gen-swagger
