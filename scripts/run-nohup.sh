#!/bin/bash

# chkconfig: - 85 15
# description: serverNameExample

serverName="serverNameExample"
cmdStr="cmd/${serverName}/${serverName}"
mainGoFile="cmd/${serverName}/main.go"
configFile=""

function checkResult() {
    result=$1
    if [ ${result} -ne 0 ]; then
        exit ${result}
    fi
}

function getPid() {
    # Get the pid from the process name
    ID=`ps -ef | grep "${cmdStr}" | grep -v "$0" | grep -v "grep" | awk '{print $2}'`
    if [ -n "$ID" ]; then
        echo $ID
        return 0
    fi

    echo ""
    return 1
}

function stopService(){
    local NAME=$1
    local pid=$(getPid)

    if [ -n "${pid}" ]; then
        kill -9 ${pid}
        checkResult $?
        echo "Stopped ${NAME} service successfully, process ID=${pid}"
    else
        echo "Service ${NAME} is not running"
    fi
}

function startService() {
    local NAME=$1

    # Check if service is already running
    local existingPid=$(getPid)
    if [ -n "${existingPid}" ]; then
        echo "Service ${NAME} is already running, process ID=${existingPid}"
        return 1
    fi

    sleep 0.2
    echo "Building ${NAME} service..."
    go build -o ${cmdStr} ${mainGoFile}
    checkResult $?

    # running server, append log to file
    echo "Starting ${NAME} service..."
    if [ -n "${configFile}" ] && [ -f "${configFile}" ]; then
        nohup ${cmdStr} -c ${configFile} >> ${NAME}.log 2>&1 &
    else
        nohup ${cmdStr} >> ${NAME}.log 2>&1 &
    fi

    # Get the PID of the last background process
    local pid=$!

    # Use for loop to check service status 5 times with 1 second delay each
    local started=0
    for i in {1..5}; do
        sleep 1
        local currentPid=$(getPid)
        if [ -n "${currentPid}" ]; then
            started=1
            echo "Started the ${NAME} service successfully, process ID=${currentPid}"
            break
        else
            echo "Checking service status... attempt ${i}/5"
        fi
    done

    if [ ${started} -eq 0 ]; then
        echo "Failed to start ${NAME} service after 5 attempts"
        return 1
    fi
    return 0
}

function restartService() {
    local NAME=$1
    echo "Restarting ${NAME} service..."
    stopService ${NAME}
    sleep 2
    startService ${NAME}
}

function statusService() {
    local NAME=$1
    local pid=$(getPid)

    if [ -n "${pid}" ]; then
        echo "Service ${NAME} is running, process ID=${pid}"
    else
        echo "Service ${NAME} is not running"
    fi
}

function showUsage() {
    echo "Usage: $0 {start|stop|restart|status} [config]"
    echo "  start      Start the service"
    echo "  stop       Stop the service"
    echo "  restart    Restart the service"
    echo "  status     Show service status"
    echo "  config     Optional configuration file path"
}

# Main script logic
case "$1" in
    start)
        if [ -n "$2" ]; then
            configFile=$2
        fi
        startService ${serverName}
        ;;
    stop)
        stopService ${serverName}
        ;;
    restart)
        if [ -n "$2" ]; then
            configFile=$2
        fi
        restartService ${serverName}
        ;;
    status)
        statusService ${serverName}
        ;;
    *)
        showUsage
        exit 1
        ;;
esac
