#!/bin/bash

function checkResult() {
    result=$1
    if [ ${result} -ne 0 ]; then
        exit ${result}
    fi
}

echo "go mod tidy"
go mod tidy
checkResult $?
gofmt -s -w .

# get config address from serverNameExample_mixExample.yml
configHost=$(bash scripts/parseYaml.sh configs/serverNameExample_mixExample.yml '.app.host')
if [ "${configHost}" = "127.0.0.1" ]; then
  configHost="localhost"
fi
configPort=$(bash scripts/parseYaml.sh configs/serverNameExample_mixExample.yml '.http.port')
configAddr="${configHost}:${configPort}"
enableMode=$(bash scripts/parseYaml.sh configs/serverNameExample_mixExample.yml '.http.tls.enableMode')
schemes="http"
if [ "$enableMode" != "" ]; then
  schemes="https"
fi

# get swagger address from main.go
swaggerAddr=$(grep -E '^[[:space:]]*//[[:space:]]*@host' "cmd/serverNameExample_mixExample/main.go" | awk '{print $3}')
if [[ -z "${swaggerAddr}" ]]; then
  swaggerAddr="localhost:8080"
fi
if [ "${configAddr}" != "${swaggerAddr}" ];then
  sed -i "s/${swaggerAddr}/${configAddr}/g" cmd/serverNameExample_mixExample/main.go
fi

# generate api docs
swag init -g cmd/serverNameExample_mixExample/main.go
checkResult $?

# modify duplicate numbers and error codes
sponge patch modify-dup-num --dir=internal/ecode
sponge patch modify-dup-err-code --dir=internal/ecode
# handle swagger.json
sponge web swagger --enable-to-openapi3 --file=docs/swagger.json > /dev/null

colorGreen='\033[1;32m'
colorCyan='\033[1;36m'
highBright='\033[1m'
markEnd='\033[0m'

echo ""
echo -e "${highBright}Tip:${markEnd} start the service with ${colorCyan}make run${markEnd}, and open ${colorCyan}${schemes}://${configAddr}/swagger/index.html${markEnd} to explore the Swagger API docs."
echo ""
echo -e "${colorGreen}generated api docs done.${markEnd}"
echo ""
