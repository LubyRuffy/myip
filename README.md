# myip
互联网侧调试访问者信息的工具

## 背景
我们经常想知道：
- 我们的公网IP是什么
- 我们使用的代理是否是高匿（是否带有X-Forwarded-For或者X-Real-IP等，是否有X-Via等）
- 我们修改Header头的程序是否生效，比如User-Agent等
- 我们想要获取一些更多的IP属性，比如国家信息

这时候我们又不想自己取开发代码写脚本，成本高收益低，而现有的平台又很有可能不稳定（比如某一天需要登录才能使用）。我实在不觉得屁大一点事就要登录甚至收费，所以还不如自己做一个，放到互联网上给大家使用。

可以访问 http://ip.bmh.im 来验证

## Feature
- [x] / 首页只返回IP的字符串
- [x] /c或者/country json格式，ip,country 
- [x] /geo 行格式，ip,upstream,country,province,city 
- [x] /ip json格式，带有ip,country,upstream（上一来源如果有的话）
- [x] /h或者/header json格式，返回ip,country,upstream,header
- [ ] 支持ipv6？
- [ ] 支持代理验证？

## 运行
```shell
go install github.com/LubyRuffy/myip@latest
`go env GOPATH`/bin/myip 
```

绑定到80端口需要root权限
```shell
cd `go env GOPATH`/bin
sudo ./myip -addr :80 
```

## 致谢
目前看起来，不用登陆，还能免费下载和使用的ip库，只剩下db-ip了。