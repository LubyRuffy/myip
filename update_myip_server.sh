#!/usr/bin/bash
# update
go install github.com/LubyRuffy/myip@latest
# restart
sudo pkill myip | sudo nohup sh -c "`go env GOPATH`/bin/myip -addr :80" 1>myip_out.txt 2>myip_err.txt &
