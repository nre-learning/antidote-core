# SHELL=/bin/bash

TARGET_VERSION ?= latest

all: compile

compile:

	@# First need to ensure all our module dependencies are vendored, and then copy non-go deps like proto files
	@go mod vendor
	@rm -rf vendor/github.com/grpc-ecosystem/grpc-gateway/third_party || true
	@# TODO(mierdin): Predict version
	@cp -r $$GOPATH/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.11.1/third_party/ vendor/github.com/grpc-ecosystem/grpc-gateway/

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

	# Use extldflags and linkmode args shown below when building for Docker using scratch image
	@go install -ldflags "-linkmode external -extldflags -static" ./cmd/...
	@#go install ./cmd/...

docker:
	docker build -t antidotelabs/antidote-core:$(TARGET_VERSION) .
	docker push antidotelabs/antidote-core:$(TARGET_VERSION)

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


# Compile binaries from vendored libs
# (This is important so that we keep the version of our tools and our vendored libraries identical)
install_bins:

	cd $$GOPATH/src/github.com/nre-learning/antidote-core/vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway/ && go install ./...
	cd $$GOPATH/src/github.com/nre-learning/antidote-core/vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/ && go install ./...
	cd $$GOPATH/src/github.com/nre-learning/antidote-core/vendor/github.com/golang/protobuf/protoc-gen-go/ && go install ./...
