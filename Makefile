# SHELL=/bin/bash

all: compile

clean:
	rm -f $(GOPATH)/bin/syringed
	rm -f $(GOPATH)/bin/syrctl

compile:

	@echo "Generating protobuf code..."
	@rm -f pkg/ui/data/swagger/datafile.go
	@rm -f /tmp/datafile.go
	@rm -f cmd/syringed/buildinfo.go
	@rm -rf api/exp/generated/ && mkdir -p api/exp/generated/
	@./compile-proto.sh

	@# Adding equivalent YAML tags so we can import lesson definitions into protobuf-created structs
	@sed -i'.bak' -e 's/\(protobuf.*json\):"\([^,]*\)/\1:"\2,omitempty" yaml:"\l\2/' api/exp/generated/lessondef.pb.go
	@rm -f api/exp/generated/lessondef.pb.go.bak

	@echo "Generating swagger definitions..."
	@go generate ./api/exp/swagger/
	@hack/build-ui.sh

	@echo "Generating build info file..."
	@hack/gen-build-info.sh

	@echo "Compiling syringe binaries..."

ifeq ($(shell uname), Darwin)
	@go install ./cmd/...
else
	@go install -ldflags "-linkmode external -extldflags -static" ./cmd/...
endif

docker:
	docker build -t antidotelabs/syringe .
	docker push antidotelabs/syringe:latest

test: 
	@go test ./api/... -cover
	@go test ./cmd/... -cover
	@go test ./config/... -cover
	@go test ./scheduler/... -cover

	@# This is causing problems.
	@#@go test ./pkg/... -cover


update:
	# Run the below to clear everything out and start from scratch
	# rm -rf ~/.glide && rm -rf vendor/ && rm -f glide.lock

	glide up -v

gengo:
	# You should only need to run this if the CRD API definitions change. Make sure you re-commit the changes once done.
	# https://blog.openshift.com/kubernetes-deep-dive-code-generation-customresources/

	git clone https://github.com/kubernetes/code-generator vendor/k8s.io/code-generator/ || true
	cd vendor/k8s.io/code-generator && git pull && cd ../../../

	rm -rf pkg/client/clientset && rm -rf pkg/client/informers && rm -rf pkg/client/listers
	
	vendor/k8s.io/code-generator/generate-groups.sh all \
	github.com/nre-learning/syringe/pkg/client \
	github.com/nre-learning/syringe/pkg/apis \
	k8s.cni.cncf.io:v1

install_bins_linux:

	@curl -L https://github.com/grpc-ecosystem/grpc-gateway/releases/download/v1.5.1/protoc-gen-grpc-gateway-v1.5.1-linux-x86_64 -o $$GOPATH/bin/protoc-gen-grpc-gateway && chmod +x $$GOPATH/bin/protoc-gen-grpc-gateway
	@curl -L https://github.com/grpc-ecosystem/grpc-gateway/releases/download/v1.5.1/protoc-gen-swagger-v1.5.1-linux-x86_64 -o $$GOPATH/bin/protoc-gen-swagger && chmod +x $$GOPATH/bin/protoc-gen-swagger

install_bins_mac:

	@curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v3.2.0/protoc-3.2.0-osx-x86_64.zip && rm -rf protoc3 && unzip protoc-3.2.0-osx-x86_64.zip -d protoc3 && chmod +x protoc3/bin/* && sudo mv protoc3/bin/* /usr/local/bin && sudo mv protoc3/include/* /usr/local/include/
	@curl -L https://github.com/grpc-ecosystem/grpc-gateway/releases/download/v1.5.1/protoc-gen-grpc-gateway-v1.5.1-darwin-x86_64 -o $$GOPATH/bin/protoc-gen-grpc-gateway && chmod +x $$GOPATH/bin/protoc-gen-grpc-gateway
	@curl -L https://github.com/grpc-ecosystem/grpc-gateway/releases/download/v1.5.1/protoc-gen-swagger-v1.5.1-darwin-x86_64 -o $$GOPATH/bin/protoc-gen-swagger && chmod +x $$GOPATH/bin/protoc-gen-swagger
