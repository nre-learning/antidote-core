declare -a protos=(
    "kubelab.proto"
    "lessondef.proto"
    "livelesson.proto"
    "syringeinfo.proto"
)

for i in "${protos[@]}"
do
    echo "Compiling protobufs for $i..."
    protoc -I api/exp/definitions/ -I./api/exp/definitions \
        -I api/exp/definitions/ \
        api/exp/definitions/"$i" \
            -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
            -I$GOPATH/src/github.com/lyft/protoc-gen-validate \
        --go_out=plugins=grpc:api/exp/generated/ \
        --grpc-gateway_out=logtostderr=true,allow_delete_body=true:api/exp/generated/ \
        --validate_out=lang=go:api/exp/generated/ \
        --swagger_out=logtostderr=true,allow_delete_body=true:api/exp/definitions/
done
