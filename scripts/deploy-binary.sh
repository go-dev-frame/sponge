#!/usr/bin/expect

set serviceName "serverNameExample"

if {$argc < 3} {
    puts "Usage: $argv0 username password ipAddr"
    exit 1
}

# parameters
set username [lindex $argv 0]
set password [lindex $argv 1]
set ipAddr [lindex $argv 2]

set timeout 30

spawn scp -r ./${serviceName}-binary.tar.gz ${username}@${ipAddr}:/tmp/
#expect "*yes/no*"
#send  "yes\r"
expect "*password:*"
send  "${password}\r"
expect eof

spawn ssh ${username}@${ipAddr}
#expect "*yes/no*"
#send  "yes\r"
expect "*password:*"
send  "${password}\r"

# execute a command or script
expect "*${username}@*"
send "cd /tmp && tar zxvf ${serviceName}-binary.tar.gz\r"
expect "*${username}@*"
send "bash /tmp/${serviceName}-binary/deploy.sh\r"

# logging out of a session
expect "*${username}@*"
send "exit\r"

expect eof
