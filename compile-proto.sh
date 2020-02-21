#!/bin/bash

declare -a protos=(
    'collection.proto'
    'lesson.proto'
    'livelesson.proto'
    'syringeinfo.proto'
    'curriculum.proto'
    'syringeinfo.proto'
    'image.proto'
    'livesession.proto'
);

for i in "${protos[@]}"
do
    echo "Compiling protobufs for $i..."
    protoc -I api/exp/definitions/ -I./api/exp/definitions \
        -I api/exp/definitions/ \
        api/exp/definitions/"$i" \
            -I$GOPATH/src/github.com/nre-learning/syringe/vendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
        --go_out=plugins=grpc:api/exp/generated/ \
        --grpc-gateway_out=logtostderr=true,allow_delete_body=true:api/exp/generated/ \
        --swagger_out=logtostderr=true,allow_delete_body=true:api/exp/definitions/
done
