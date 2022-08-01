#!/usr/bin/bash
# update
go install github.com/LubyRuffy/myip@latest
# restart
# ps axu | grep myip | grep -v grep | awk '{print $2}' | sudo xargs kill
sudo pkill myip
sudo nohup sh -c "`go env GOPATH`/bin/myip -addr :80" 1>myip_out.txt 2>myip_err.txt &
