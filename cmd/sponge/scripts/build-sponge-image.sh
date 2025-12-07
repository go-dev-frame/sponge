#!/bin/bash

TAG=$1
if [ "X${TAG}" = "X" ];then
    echo "image tag cannot be empty, example: ./build-sponge-image.sh v1.5.8"
    exit 1
fi

function rmFile() {
    sFile=$1
    if [ -e "${sFile}" ]; then
        rm -rf ${sFile}
    fi
}

function checkResult() {
    result=$1
    if [ ${result} -ne 0 ]; then
        exit ${result}
    fi
}

# download the specified version of the sponge binary file
binaryFile="sponge_${TAG#v}_linux_amd64.zip"
rmFile ${binaryFile}
wget https://github.com/go-dev-frame/sponge/releases/download/${TAG}/${binaryFile}
checkResult $?
unzip -o -q ${binaryFile}
rmFile ${binaryFile} && rmFile LICENSE && rmFile README.md

# download the specified version of the sponge template code
codeFile="${TAG}.zip"
rmFile ${codeFile}
wget https://github.com/go-dev-frame/sponge/archive/refs/tags/${codeFile}
checkResult $?
unzip -o -q ${codeFile}
mv sponge-${TAG#v} .sponge
echo ${TAG} > .sponge/.github/version
rmFile ${codeFile} && rm -rf .sponge/cmd/sponge

# compressing binary file
upx -9 sponge
checkResult $?

echo "docker build -t zhufuyi/sponge:${TAG}  ."
docker build -t zhufuyi/sponge:${TAG}  .
checkResult $?

rmFile sponge
rm -rf .sponge

# delete none image
noneImages=$(docker images | grep "<none>" | awk '{print $3}')
if [ "X${noneImages}" != "X" ]; then
  docker rmi ${noneImages} > /dev/null 2>&1 || true
fi
