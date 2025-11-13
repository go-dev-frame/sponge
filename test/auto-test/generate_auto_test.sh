#!/bin/bash

runCase=$1

run_script() {
  script="files/${1}.sh"
  if [[ -f "$script" ]]; then
    echo ">>> Running $script"
    bash "$script"
    echo ">>> Done: $script"
    echo
  else
    echo "!!! Script not found: $script"
  fi
}

case "$runCase" in
  1_web_http)
    run_script "1_web_http"
    ;;
  2_micro_grpc)
    run_script "2_micro_grpc"
    ;;
  3_micro_grpc_http)
    run_script "3_micro_grpc_http"
    ;;
  4_web_http_pb)
    run_script "4_web_http_pb"
    ;;
  5_micro_grpc_pb)
    run_script "5_micro_grpc_pb"
    ;;
  6_micro_grpc_http_pb)
    run_script "6_micro_grpc_http_pb"
    ;;
  7_micro_grpc_gateway_pb)
    run_script "7_micro_grpc_gateway_pb"
    ;;
  "" )
    echo ">>> Running all scripts..."
    for i in {1..7}; do
      run_script "${i}_$(ls files | sed -n "${i}p" | cut -d'_' -f2- | sed 's/.sh$//')"
    done
    ;;
  *)
    echo "Invalid argument: $runCase"
    echo "Usage: bash auto.sh [1_web_http|2_micro_grpc|3_micro_grpc_http|4_web_http_pb|5_micro_grpc_pb|6_micro_grpc_http_pb|7_micro_grpc_gateway_pb]"
    ;;
esac
