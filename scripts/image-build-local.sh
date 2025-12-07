#!/bin/bash

# build the image for local docker, using the binaries, if you want to reduce the size of the image,
# use upx to compress the binaries before building the image.

serverName="serverNameExample_mixExample"
# image name of the service, prohibit uppercase letters in names.
IMAGE_NAME="project-name-example/server-name-example"
# Dockerfile file directory
DOCKERFILE_PATH="scripts/build"
DOCKERFILE="${DOCKERFILE_PATH}/Dockerfile"

mv -f cmd/${serverName}/${serverName} ${DOCKERFILE_PATH}/${serverName}

# todo generate image-build-local code for http or grpc here
# delete the templates code start

# install grpc-health-probe, for health check of grpc service
rootDockerFilePath=$(pwd)/${DOCKERFILE_PATH}
go install github.com/grpc-ecosystem/grpc-health-probe@v0.4.12
cd $GOPATH/pkg/mod/github.com/grpc-ecosystem/grpc-health-probe@v0.4.12 \
    && go mod download \
    && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "all=-s -w" -o "${rootDockerFilePath}/grpc_health_probe"
cd -

# compressing binary file
#cd ${DOCKERFILE_PATH}
#upx -9 ${serverName}
#upx -9 grpc_health_probe
#cd -

mkdir -p ${DOCKERFILE_PATH}/configs && cp -f configs/${serverName}.yml ${DOCKERFILE_PATH}/configs/
echo "docker build -f ${DOCKERFILE} -t ${IMAGE_NAME}:latest ${DOCKERFILE_PATH}"
docker build -f ${DOCKERFILE} -t ${IMAGE_NAME}:latest ${DOCKERFILE_PATH}

if [ -f "${DOCKERFILE_PATH}/grpc_health_probe" ]; then
    rm -f ${DOCKERFILE_PATH}/grpc_health_probe
fi

# delete the templates code end

if [ -f "${DOCKERFILE_PATH}/${serverName}" ]; then
    rm -f ${DOCKERFILE_PATH}/${serverName}
fi

if [ -d "${DOCKERFILE_PATH}/configs" ]; then
    rm -rf ${DOCKERFILE_PATH}/configs
fi

# delete none image
noneImages=$(docker images | grep "${IMAGE_NAME}" | grep "<none>" | awk '{print $3}')
if [ "X${noneImages}" != "X" ]; then
  docker rmi ${noneImages} > /dev/null 2>&1 || true
fi
exit 0
