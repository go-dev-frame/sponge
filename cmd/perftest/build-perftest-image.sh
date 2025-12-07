#!/bin/bash

TAG=$1
if [ -z "$TAG" ];then
    echo "image tag cannot be empty, example: ./build-perftest-image.sh v1.0.0"
    exit 1
fi

function rmFile() {
    sFile=$1
    if [ -e "${sFile}" ]; then
        rm -rf "${sFile}"
    fi
}

function checkResult() {
    result=$1
    if [ ${result} -ne 0 ]; then
        exit ${result}
    fi
}

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "all=-s -w"
checkResult $?

# compressing binary file
if command -v upx >/dev/null 2>&1; then
    upx -9 perftest
else
    echo "upx not found, skipping compression"
fi

echo "docker build -t zhufuyi/perftest:${TAG}  ."
docker build -t zhufuyi/perftest:${TAG}  .
echo "docker tag zhufuyi/perftest:${TAG} zhufuyi/perftest:latest"
docker tag zhufuyi/perftest:${TAG} zhufuyi/perftest:latest
checkResult $?

rmFile perftest

# delete none image
noneImages=$(docker images --filter "dangling=true" -q | grep "${zhufuyi/perftest}")
if [ -n "$noneImages" ]; then
    docker rmi $noneImages > /dev/null 2>&1 || true
fi

# push image to docker hub
#docker push zhufuyi/perftest:${TAG}
#docker push zhufuyi/perftest:latest
