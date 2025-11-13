#!/bin/bash

mysqlDsn="root:123456@(192.168.3.37:3306)/account"
mysqlTable1="user"
mysqlTable2="user_order"
mysqlTable3="user_str"

postgresqlDsn="root:123456@(192.168.3.37:5432)/account"
postgresqlTable1="user"
postgresqlTable2="user_order"
postgresqlTable3="user_str"

sqliteDsn="../sql/sqlite/sponge.db"
sqliteTable1="user"
sqliteTable2="user_order"
sqliteTable3="user_str"

mongodbDsn="root:123456@(192.168.3.37:27017)/account"
mongodbCollection1="user"
mongodbCollection2="user_order"
mongodbCollection3="people"

colorCyan='\e[1;36m'
colorGreen='\e[1;32m'
colorRed='\e[1;31m'
markEnd='\e[0m'

httpProtobufFile="files/user_http.proto"
grpcProtobufFile="files/user_rpc.proto"
mixProtobufFile="files/user_hybrid.proto"
grpcGwProtobufFile="files/user_gw.proto"
mixedProtobufFile="files/mixed.proto"

isOnlyGenerateCode="false"

isExtended="false"
if [ "$1" == "true" ]; then
  isExtended="true"
else
  isExtended="false"
fi

function checkResult() {
    result=$1
    if [ ${result} -ne 0 ]; then
        exit ${result}
    fi
}

serverRunSuccess="false"

function checkAndPrintTestResult() {
  if [ "$serverRunSuccess" == "true" ]; then
    echo -e "\n\n${colorGreen}--------------------- [${serverDir}] test result: passed ---------------------${markEnd}\n"
  fi
  serverRunSuccess="false" # reset serverRunSuccess
}

function printTestFailed() {
  echo -e "\n\n${colorRed}--------------------- [${serverDir}] test result: failed ---------------------${markEnd}\n"
}

function stopService() {
  local name=$1
  if [ "$name" == "" ]; then
    echo "name cannot be empty"
    return 1
  fi

  local processMark="./cmd/$name"
  pid=$(ps -ef | grep "${processMark}" | grep -v grep | awk '{print $2}')
  if [ "${pid}" != "" ]; then
      kill -9 ${pid}
      return 0
  fi

  return 1
}

function checkServiceStarted() {
  local name=$1
  local serverDir=$2
  if [ "$name" == "" ]; then
    echo "name cannot be empty"
    return 1
  fi

  local processMark="./cmd/$name"
  local timeCount=0
  # waiting for service to start
  while true; do
    sleep 1
    pid=$(ps -ef | grep "${processMark}" | grep -v grep | awk '{print $2}')
    if [ "${pid}" != "" ]; then
        serverRunSuccess="true"
        break
    fi
    (( timeCount++ ))
    if (( timeCount >= 20 )); then
      printTestFailed
      return 1
    fi
  done
}

function runningHTTPService() {
  local name=$1
  local serverDir=$2
  if [ "$name"x = x ];then
    echo "server name cannot be empty"
    return 1
  fi

  make docs
  checkResult $?
  echo "startup service $name"
  make run &
  checkServiceStarted $name $serverDir
  checkResult $?
  sleep 1
  stopService $name
  checkResult $?
  checkAndPrintTestResult
}

function runningProtoService() {
  local name=$1
  local serverDir=$2
  if [ "$name"x = x ];then
    echo "server name cannot be empty"
    return 1
  fi

  make proto
  checkResult $?
  echo -e "startup service $name"
  make run &
  checkServiceStarted $name $serverDir
  checkResult $?
  sleep 1
  stopService $name
  checkResult $?
  checkAndPrintTestResult
}

# -------------------------------------------------------------------------------------------

function generate_http_mysql() {
  local serverName="user"
  local outDir="multi-01-http-mysql"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge web http --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable1 --embed=true --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge web http --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable1 --embed=true --extended-api=$isExtended --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge web handler --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable2,$mysqlTable3 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge web handler --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable2,$mysqlTable3 --extended-api=$isExtended --out=$outDir
    checkResult $?
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  runningHTTPService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}

function generate_http_postgresql() {
  local serverName="user"
  local outDir="multi-02-http-postgresql"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge web http --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=postgresql --db-dsn=$postgresqlDsn --db-table=$postgresqlTable1 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge web http --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=postgresql --db-dsn=$postgresqlDsn --db-table=$postgresqlTable1 --extended-api=$isExtended --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge web handler --db-driver=postgresql --db-dsn=$postgresqlDsn --db-table=$postgresqlTable2,$postgresqlTable3 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge web handler --db-driver=postgresql --db-dsn=$postgresqlDsn --db-table=$postgresqlTable2,$postgresqlTable3 --extended-api=$isExtended --out=$outDir
    checkResult $?
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  runningHTTPService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}

function generate_http_sqlite() {
  local serverName="user"
  local outDir="multi-03-http-sqlite"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge web http --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=sqlite --db-dsn=$sqliteDsn --db-table=$sqliteTable1 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge web http --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=sqlite --db-dsn=$sqliteDsn --db-table=$sqliteTable1 --extended-api=$isExtended --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge web handler --db-driver=sqlite --db-dsn=$sqliteDsn --db-table=$sqliteTable2,$sqliteTable3 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge web handler --db-driver=sqlite --db-dsn=$sqliteDsn --db-table=$sqliteTable2,$sqliteTable3 --extended-api=$isExtended --out=$outDir
    checkResult $?

    sed -E -i 's/\\\\sql\\\\/\\\\\.\.\\\\\sql\\\\/g' ${outDir}/configs/${serverName}.yml
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  runningHTTPService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}

function generate_http_mongodb() {
  local serverName="user"
  local outDir="multi-04-http-mongodb"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge web http --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=mongodb --db-dsn=$mongodbDsn --db-table=$mongodbCollection1 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge web http --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=mongodb --db-dsn=$mongodbDsn --db-table=$mongodbCollection1 --extended-api=$isExtended --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge web handler --db-driver=mongodb --db-dsn=$mongodbDsn --db-table=$mongodbCollection2,$mongodbCollection3 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge web handler --db-driver=mongodb --db-dsn=$mongodbDsn --db-table=$mongodbCollection2,$mongodbCollection3 --extended-api=$isExtended --out=$outDir
    checkResult $?
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  runningHTTPService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}

# ---------------------------------------------------------------

function generate_grpc_mysql() {
  local serverName="user"
  local outDir="multi-05-grpc-mysql"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge micro rpc --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable1 --embed=true --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro rpc --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable1 --embed=true --extended-api=$isExtended --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge micro service --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable2,$mysqlTable3 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro service --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable2,$mysqlTable3 --extended-api=$isExtended --out=$outDir
    checkResult $?
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  runningProtoService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}

function generate_grpc_postgresql() {
  local serverName="user"
  local outDir="multi-06-grpc-postgresql"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge micro rpc --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=postgresql --db-dsn=$postgresqlDsn --db-table=$postgresqlTable1 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro rpc --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=postgresql --db-dsn=$postgresqlDsn --db-table=$postgresqlTable1 --extended-api=$isExtended --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge micro service --db-driver=postgresql --db-dsn=$postgresqlDsn --db-table=$postgresqlTable2,$postgresqlTable3 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro service --db-driver=postgresql --db-dsn=$postgresqlDsn --db-table=$postgresqlTable2,$postgresqlTable3 --extended-api=$isExtended --out=$outDir
    checkResult $?
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  runningProtoService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}

function generate_grpc_sqlite() {
  local serverName="user"
  local outDir="multi-07-grpc-sqlite"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge micro rpc --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=sqlite --db-dsn=$sqliteDsn --db-table=$sqliteTable1 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro rpc --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=sqlite --db-dsn=$sqliteDsn --db-table=$sqliteTable1 --extended-api=$isExtended --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge micro service --db-driver=sqlite --db-dsn=$sqliteDsn --db-table=$sqliteTable2,$sqliteTable3 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro service --db-driver=sqlite --db-dsn=$sqliteDsn --db-table=$sqliteTable2,$sqliteTable3 --extended-api=$isExtended --out=$outDir
    checkResult $?

    sed -E -i 's/\\\\sql\\\\/\\\\\.\.\\\\\sql\\\\/g' ${outDir}/configs/${serverName}.yml
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  runningProtoService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}

function generate_grpc_mongodb() {
  local serverName="user"
  local outDir="multi-08-grpc-mongodb"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge micro rpc --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=mongodb --db-dsn=$mongodbDsn --db-table=$mongodbCollection1 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro rpc --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=mongodb --db-dsn=$mongodbDsn --db-table=$mongodbCollection1 --extended-api=$isExtended --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge micro service --db-driver=mongodb --db-dsn=$mongodbDsn --db-table=$mongodbCollection2,$mongodbCollection3 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro service --db-driver=mongodb --db-dsn=$mongodbDsn --db-table=$mongodbCollection2,$mongodbCollection3 --extended-api=$isExtended --out=$outDir
    checkResult $?
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  runningProtoService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}

# ---------------------------------------------------------------

function generate_grpc_http_mysql() {
  local serverName="user"
  local outDir="multi-09-grpc-http-mysql"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge micro grpc-http --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable1 --embed=true --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro grpc-http --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable1 --embed=true --extended-api=$isExtended --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge micro service-handler --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable2,$mysqlTable3 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro service-handler --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable2,$mysqlTable3 --extended-api=$isExtended --out=$outDir
    checkResult $?
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  runningProtoService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}

function generate_grpc_http_postgresql() {
  local serverName="user"
  local outDir="multi-10-grpc-http-postgresql"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge micro grpc-http --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=postgresql --db-dsn=$postgresqlDsn --db-table=$postgresqlTable1 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro grpc-http --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=postgresql --db-dsn=$postgresqlDsn --db-table=$postgresqlTable1 --extended-api=$isExtended --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge micro service-handler --db-driver=postgresql --db-dsn=$postgresqlDsn --db-table=$postgresqlTable2,$postgresqlTable3 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro service-handler --db-driver=postgresql --db-dsn=$postgresqlDsn --db-table=$postgresqlTable2,$postgresqlTable3 --extended-api=$isExtended --out=$outDir
    checkResult $?
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  runningProtoService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}

function generate_grpc_http_sqlite() {
  local serverName="user"
  local outDir="multi-11-grpc-http-sqlite"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge micro grpc-http --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=sqlite --db-dsn=$sqliteDsn --db-table=$sqliteTable1 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro grpc-http --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=sqlite --db-dsn=$sqliteDsn --db-table=$sqliteTable1 --extended-api=$isExtended --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge micro service-handler --db-driver=sqlite --db-dsn=$sqliteDsn --db-table=$sqliteTable2,$sqliteTable3 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro service-handler --db-driver=sqlite --db-dsn=$sqliteDsn --db-table=$sqliteTable2,$sqliteTable3 --extended-api=$isExtended --out=$outDir
    checkResult $?

    sed -E -i 's/\\\\sql\\\\/\\\\\.\.\\\\\sql\\\\/g' ${outDir}/configs/${serverName}.yml
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  runningProtoService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}

function generate_grpc_http_mongodb() {
  local serverName="user"
  local outDir="multi-12-grpc-http-mongodb"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge micro grpc-http --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=mongodb --db-dsn=$mongodbDsn --db-table=$mongodbCollection1 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro grpc-http --server-name=$serverName --module-name=$serverName --project-name=edusys --db-driver=mongodb --db-dsn=$mongodbDsn --db-table=$mongodbCollection1 --extended-api=$isExtended --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge micro service-handler --db-driver=mongodb --db-dsn=$mongodbDsn --db-table=$mongodbCollection2,$mongodbCollection3 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro service-handler --db-driver=mongodb --db-dsn=$mongodbDsn --db-table=$mongodbCollection2,$mongodbCollection3 --extended-api=$isExtended --out=$outDir
    checkResult $?
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  runningProtoService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}

# ---------------------------------------------------------------

function generate_http_pb_mysql() {
  local serverName="user"
  local outDir="multi-13-http-pb-mysql"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge web http-pb --server-name=$serverName --module-name=$serverName --project-name=edusys --protobuf-file=${httpProtobufFile} --out=$outDir ${markEnd}"
    sponge web http-pb --server-name=$serverName --module-name=$serverName --project-name=edusys --protobuf-file=${httpProtobufFile} --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge web handler-pb--db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable1 --embed=true --extended-api=$isExtended  --out=$outDir ${markEnd}"
    sponge web handler-pb --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable1 --embed=true --extended-api=$isExtended --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge web handler-pb --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable2,$mysqlTable3 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge web handler-pb --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable2,$mysqlTable3 --extended-api=$isExtended --out=$outDir
    checkResult $?

    mysqlDsnTmp=$(echo "$mysqlDsn" | sed -E 's/\(/\\\(/g' | sed -E 's/\)/\\\)/g' | sed -E 's/\//\\\//g')
    sed -E -i "s/root:123456@\(192.168.3.37:3306\)\/account/${mysqlDsnTmp}/g" ${outDir}/configs/${serverName}.yml
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  make patch TYPE=types-pb
  make patch TYPE=init-mysql
  runningProtoService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}

function generate_http_pb_mongodb() {
  local serverName="user"
  local outDir="multi-14-http-pb-mongodb"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge web http-pb --server-name=$serverName --module-name=$serverName --project-name=edusys --protobuf-file=${httpProtobufFile} --out=$outDir ${markEnd}"
    sponge web http-pb --server-name=$serverName --module-name=$serverName --project-name=edusys --protobuf-file=${httpProtobufFile} --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge web handler-pb --db-driver=mongodb --db-dsn=$mongodbDsn --db-table=$mongodbCollection1,$mongodbCollection2,$mongodbCollection3 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge web handler-pb --db-driver=mongodb --db-dsn=$mongodbDsn --db-table=$mongodbCollection1,$mongodbCollection2,$mongodbCollection3 --extended-api=$isExtended --out=$outDir
    checkResult $?

    sed -E -i 's/\"mysql\"/\"mongodb\"/g' ${outDir}/configs/${serverName}.yml
    sed -E -i 's/mysql:/mongodb:/g' ${outDir}/configs/${serverName}.yml
    mongodbDsnTmp=$(echo "$mongodbDsn" | sed -E 's/\(/\\\(/g' | sed -E 's/\)/\\\)/g' | sed -E 's/\//\\\//g')
    sed -E -i "s/root:123456@\(192.168.3.37:3306\)\/account/${mongodbDsnTmp}/g" ${outDir}/configs/${serverName}.yml
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  make patch TYPE=types-pb
  make patch TYPE=init-mongodb
  runningProtoService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}

# ---------------------------------------------------------------

function generate_grpc_pb_mysql() {
  local serverName="user"
  local outDir="multi-15-grpc-pb-mysql"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge micro rpc-pb --server-name=$serverName --module-name=$serverName --project-name=edusys --protobuf-file=${grpcProtobufFile} --out=$outDir ${markEnd}"
    sponge micro rpc-pb --server-name=$serverName --module-name=$serverName --project-name=edusys --protobuf-file=${grpcProtobufFile} --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge micro service --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable1 --embed=true --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro service --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable1 --embed=true --extended-api=$isExtended --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge micro service --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable2,$mysqlTable3 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro service --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable2,$mysqlTable3 --extended-api=$isExtended --out=$outDir
    checkResult $?

    mysqlDsnTmp=$(echo "$mysqlDsn" | sed -E 's/\(/\\\(/g' | sed -E 's/\)/\\\)/g' | sed -E 's/\//\\\//g')
    sed -E -i "s/root:123456@\(192.168.3.37:3306\)\/account/${mysqlDsnTmp}/g" ${outDir}/configs/${serverName}.yml
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  make patch TYPE=types-pb
  make patch TYPE=init-mysql
  runningProtoService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}

function generate_grpc_pb_mongodb() {
  local serverName="user"
  local outDir="multi-16-grpc-pb-mongodb"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge micro rpc-pb --server-name=$serverName --module-name=$serverName --project-name=edusys --protobuf-file=${grpcProtobufFile} --out=$outDir ${markEnd}"
    sponge micro rpc-pb --server-name=$serverName --module-name=$serverName --project-name=edusys --protobuf-file=${grpcProtobufFile} --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge micro service --db-driver=mongodb --db-dsn=$mongodbDsn --db-table=$mongodbCollection1,$mongodbCollection2,$mongodbCollection3 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro service --db-driver=mongodb --db-dsn=$mongodbDsn --db-table=$mongodbCollection1,$mongodbCollection2,$mongodbCollection3 --extended-api=$isExtended --out=$outDir
    checkResult $?

    sed -E -i 's/\"mysql\"/\"mongodb\"/g' ${outDir}/configs/${serverName}.yml
    sed -E -i 's/mysql:/mongodb:/g' ${outDir}/configs/${serverName}.yml
    mongodbDsnTmp=$(echo "$mongodbDsn" | sed -E 's/\(/\\\(/g' | sed -E 's/\)/\\\)/g' | sed -E 's/\//\\\//g')
    sed -E -i "s/root:123456@\(192.168.3.37:3306\)\/account/${mongodbDsnTmp}/g" ${outDir}/configs/${serverName}.yml
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  make patch TYPE=types-pb
  make patch TYPE=init-mongodb
  runningProtoService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}

# ---------------------------------------------------------------

function generate_grpc_http_pb_mysql() {
  local serverName="user"
  local outDir="multi-17-grpc-http-pb-mysql"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge micro grpc-http-pb --server-name=$serverName --module-name=$serverName --project-name=edusys --protobuf-file=${mixProtobufFile} --out=$outDir ${markEnd}"
    sponge micro grpc-http-pb --server-name=$serverName --module-name=$serverName --project-name=edusys --protobuf-file=${mixProtobufFile} --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge micro service-handler --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable1 --embed=true --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro service-handler --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable1 --embed=true --extended-api=$isExtended --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge micro service-handler --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable2,$mysqlTable3 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro service-handler  --db-driver=mysql --db-dsn=$mysqlDsn --db-table=$mysqlTable2,$mysqlTable3 --extended-api=$isExtended --out=$outDir
    checkResult $?

    mysqlDsnTmp=$(echo "$mysqlDsn" | sed -E 's/\(/\\\(/g' | sed -E 's/\)/\\\)/g' | sed -E 's/\//\\\//g')
    sed -E -i "s/root:123456@\(192.168.3.37:3306\)\/account/${mysqlDsnTmp}/g" ${outDir}/configs/${serverName}.yml
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  make patch TYPE=types-pb
  make patch TYPE=init-mysql
  runningProtoService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}

function generate_grpc_http_pb_mongodb() {
  local serverName="user"
  local outDir="multi-18-grpc-http-pb-mongodb"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge micro grpc-http-pb --server-name=$serverName --module-name=$serverName --project-name=edusys --protobuf-file=${mixProtobufFile} --out=$outDir ${markEnd}"
    sponge micro grpc-http-pb --server-name=$serverName --module-name=$serverName --project-name=edusys --protobuf-file=${mixProtobufFile} --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge micro service-handler --db-driver=mongodb --db-dsn=$mongodbDsn --db-table=$mongodbCollection1,$mongodbCollection2,$mongodbCollection3 --extended-api=$isExtended --out=$outDir ${markEnd}"
    sponge micro service-handler --db-driver=mongodb --db-dsn=$mongodbDsn --db-table=$mongodbCollection1,$mongodbCollection2,$mongodbCollection3 --extended-api=$isExtended --out=$outDir
    checkResult $?

    sed -E -i 's/\"mysql\"/\"mongodb\"/g' ${outDir}/configs/${serverName}.yml
    sed -E -i 's/mysql:/mongodb:/g' ${outDir}/configs/${serverName}.yml
    mongodbDsnTmp=$(echo "$mongodbDsn" | sed -E 's/\(/\\\(/g' | sed -E 's/\)/\\\)/g' | sed -E 's/\//\\\//g')
    sed -E -i "s/root:123456@\(192.168.3.37:3306\)\/account/${mongodbDsnTmp}/g" ${outDir}/configs/${serverName}.yml
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  make patch TYPE=types-pb
  make patch TYPE=init-mongodb
  runningProtoService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}
# ---------------------------------------------------------------

function generate_http_pb_mixed() {
  local serverName="user"
  local outDir="multi-19-http-pb-mixed"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge web http-pb --server-name=$serverName --module-name=$serverName --project-name=edusys --protobuf-file=${mixedProtobufFile} --out=$outDir ${markEnd}"
    sponge web http-pb --server-name=$serverName --module-name=$serverName --project-name=edusys --protobuf-file=${mixedProtobufFile} --out=$outDir
    checkResult $?
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  runningProtoService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}

function generate_grpc_pb_mixed() {
  local serverName="user"
  local outDir="multi-20-grpc-pb-mixed"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge micro rpc-pb --server-name=$serverName --module-name=$serverName --project-name=edusys --protobuf-file=${mixedProtobufFile} --out=$outDir ${markEnd}"
    sponge micro rpc-pb --server-name=$serverName --module-name=$serverName --project-name=edusys --protobuf-file=${mixedProtobufFile} --out=$outDir
    checkResult $?
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  runningProtoService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}

# ---------------------------------------------------------------

function generate_grpc_gw_pb_mixed() {
  local serverName="user_gw"
  local outDir="multi-21-grpc-gw-pb-mixed"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge micro rpc-gw-pb --server-name=$serverName --module-name=edusys --project-name=edusys --protobuf-file=${mixedProtobufFile} --out=$outDir ${markEnd}"
    sponge micro rpc-gw-pb --server-name=$serverName --module-name=edusys --project-name=edusys --protobuf-file=${mixedProtobufFile} --out=$outDir
    checkResult $?
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  runningProtoService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}

function generate_grpc_gw_pb() {
  local serverName="user_gw"
  local outDir="multi-22-grpc-gw-pb"
  echo -e "\n\n"
  echo -e "create service code to directory $outDir"
  if [ -d "${outDir}" ]; then
    echo -e "$outDir already exists\n\n"
  else
    echo -e "\n${colorCyan}sponge micro rpc-gw-pb --server-name=$serverName --module-name=edusys --project-name=edusys --protobuf-file=${grpcGwProtobufFile} --out=$outDir ${markEnd}"
    sponge micro rpc-gw-pb --server-name=$serverName --module-name=edusys --project-name=edusys --protobuf-file=${grpcGwProtobufFile} --out=$outDir
    checkResult $?

    echo -e "\n${colorCyan}sponge micro rpc-conn --rpc-server-name=user --out=$outDir ${markEnd}"
    sponge micro rpc-conn --rpc-server-name=user --out=$outDir
    checkResult $?
  fi

  if [ "$isOnlyGenerateCode" == "true" ]; then
    echo -e "\n\n"
    return
  fi

  cd $outDir
  make copy-proto SERVER=../multi-05-grpc-mysql
  checkResult $?
  runningProtoService $serverName $outDir
  checkResult $?
  sleep 1
  cd -
}

# -------------------------------------------------------------------------------------------

runCase=$1

run_all() {
  echo ">>> Running all cases..."
  all_cases=(
    generate_http_mysql generate_http_postgresql generate_http_sqlite generate_http_mongodb
    generate_grpc_mysql generate_grpc_postgresql generate_grpc_sqlite generate_grpc_mongodb
    generate_grpc_http_mysql generate_grpc_http_postgresql generate_grpc_http_sqlite generate_grpc_http_mongodb
    generate_http_pb_mysql generate_http_pb_mongodb
    generate_grpc_pb_mysql generate_grpc_pb_mongodb
    generate_grpc_http_pb_mysql generate_grpc_http_pb_mongodb
    generate_http_pb_mixed generate_grpc_pb_mixed generate_grpc_gw_pb_mixed generate_grpc_gw_pb
  )
  for fn in "${all_cases[@]}"; do
      echo ">>> Executing: $fn"
      $fn
      echo ">>> Done: $fn"
      echo
  done
}

case "$runCase" in
  1_http_mysql)            generate_http_mysql ;;
  2_http_postgresql)       generate_http_postgresql ;;
  3_http_sqlite)           generate_http_sqlite ;;
  4_http_mongodb)          generate_http_mongodb ;;
  5_grpc_mysql)            generate_grpc_mysql ;;
  6_grpc_postgresql)       generate_grpc_postgresql ;;
  7_grpc_sqlite)           generate_grpc_sqlite ;;
  8_grpc_mongodb)          generate_grpc_mongodb ;;
  9_grpc_http_mysql)       generate_grpc_http_mysql ;;
  10_grpc_http_postgresql) generate_grpc_http_postgresql ;;
  11_grpc_http_sqlite)     generate_grpc_http_sqlite ;;
  12_grpc_http_mongodb)    generate_grpc_http_mongodb ;;
  13_http_pb_mysql)        generate_http_pb_mysql ;;
  14_http_pb_mongodb)      generate_http_pb_mongodb ;;
  15_grpc_pb_mysql)        generate_grpc_pb_mysql ;;
  16_grpc_pb_mongodb)      generate_grpc_pb_mongodb ;;
  17_grpc_http_pb_mysql)   generate_grpc_http_pb_mysql ;;
  18_grpc_http_pb_mongodb) generate_grpc_http_pb_mongodb ;;
  19_http_pb_mixed)        generate_http_pb_mixed ;;
  20_grpc_pb_mixed)        generate_grpc_pb_mixed ;;
  21_grpc_gw_pb_mixed)     generate_grpc_gw_pb_mixed ;;
  22_grpc_gw_pb)           generate_grpc_gw_pb ;;
  "")                      run_all ;;
  *)
    echo "Invalid argument: $runCase"
    echo "Usage: bash auto.sh [1_http_mysql|2_http_postgresql|...|22_grpc_gw_pb]"
    ;;
esac
