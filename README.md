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
- [x] /geo 行格式，ip,upstream,country,province,city
- [x] /c或者/country json格式，ip,country
- [x] /ip json格式，带有ip,country,upstream（上一来源如果有的话）
- [x] /h或者/header json格式，返回ip,country,upstream,header
- [x] 支持ipv6，需要dns配置ipv6对应的AAAA记录，```dig AAAA ip.bmh.im```，客户端是ipv6的话可以直接查询
- [ ] 支持代理验证？
- [x] 所有接口支持pretty模式，默认为false，参数带有p或者pretty的情况下，会格式化输出json

## 运行

```shell
go install github.com/LubyRuffy/myip@latest
`go env GOPATH`/bin/myip 
```

### 绑定到80端口需要root权限
```shell
cd `go env GOPATH`/bin
sudo ./myip -addr :80

# 或者nohup运行
sudo nohup sh -c "`go env GOPATH`/bin/myip -addr :80" &
```

### 启动tls自动更新let's encrypt证书
```shell
cd `go env GOPATH`/bin
sudo ./myip -addr :80
sudo nohup sh -c "`go env GOPATH`/bin/myip -addr :80 -autotls -subdomains ip.bmh.im" &
```

## 测试

### 获取ip
直接连接
```shell
$ curl ip.bmh.im
61.148.1.1
```

使用代理
```shell
$ http_proxy=http://2.2.2.2 curl ip.bmh.im
2.2.2.2
```

### 获取country，json格式
```shell
$ curl ip.bmh.im/c?p
{
  "ip": "155.138.1.1",
  "country": "United States",
}
```

```shell
$ http_proxy=http://155.138.1.1 curl ip.bmh.im/c?p
{
  "ip": "155.138.1.1",
  "country": "United States",
  "upstream": "61.148.1.1",
  "upstream_country": "China"
}
```

### 获取geo，行的形式

```shell
$ curl ip.bmh.im/geo
61.148.1.1,China,Beijing,Xicheng District,Asia
```

```shell
$ http_proxy=http://155.138.1.1 curl ip.bmh.im/geo
155.138.1.1,United States,Georgia,Atlanta (Knight Park/Howell Station),North America
61.148.1.1,China,Beijing,Xicheng District,Asia
```

### 获取header，json格式
```shell
$ http_proxy=http://155.138.1.1 curl ip.bmh.im/h?p
{
  "geo": {
    "city": {
      "names": {
        "en": "Xicheng District"
      }
    },
    "continent": {
      "code": "AS",
      "geoname_id": 6255147,
      "names": {
        "de": "Asien",
        "en": "Asia",
        "es": "Asia",
        "fa": " آسیا",
        "fr": "Asie",
        "ja": "アジア大陸",
        "ko": "아시아",
        "pt-BR": "Ásia",
        "ru": "Азия",
        "zh-CN": "亚洲"
      }
    },
    "country": {
      "geoname_id": 1814991,
      "is_in_european_union": false,
      "iso_code": "CN",
      "names": {
        "de": "China, Volksrepublik",
        "en": "China",
        "es": "China",
        "fa": "چین",
        "fr": "Chine",
        "ja": "中国",
        "ko": "중국",
        "pt-BR": "China",
        "ru": "Китай",
        "zh-CN": "中国"
      }
    },
    "location": {
      "latitude": 39.9175,
      "longitude": 116.362
    },
    "subdivisions": [
      {
        "names": {
          "en": "Beijing"
        }
      }
    ]
  },
  "header": "GET /h HTTP/1.1\nUser-Agent: curl/7.83.0\nAccept: */*\nVia: 1.1 proxAtlanta01 (squid/4.11)\nX-Forwarded-For: 61.148.74.134\nCache-Control: max-age=259200\nConnection: keep-alive\nIf-Modified-Since: Sat, 30 Jul 2022 13:23:07 GMT\n",
  "ip": "155.138.1.1",
  "upstream": "61.148.1.1",
  "upstream_geo": {
    "city": {
      "names": {
        "en": "Xicheng District"
      }
    },
    "continent": {
      "code": "AS",
      "geoname_id": 6255147,
      "names": {
        "de": "Asien",
        "en": "Asia",
        "es": "Asia",
        "fa": " آسیا",
        "fr": "Asie",
        "ja": "アジア大陸",
        "ko": "아시아",
        "pt-BR": "Ásia",
        "ru": "Азия",
        "zh-CN": "亚洲"
      }
    },
    "country": {
      "geoname_id": 1814991,
      "is_in_european_union": false,
      "iso_code": "CN",
      "names": {
        "de": "China, Volksrepublik",
        "en": "China",
        "es": "China",
        "fa": "چین",
        "fr": "Chine",
        "ja": "中国",
        "ko": "중국",
        "pt-BR": "China",
        "ru": "Китай",
        "zh-CN": "中国"
      }
    },
    "location": {
      "latitude": 39.9175,
      "longitude": 116.362
    },
    "subdivisions": [
      {
        "names": {
          "en": "Beijing"
        }
      }
    ]
  }
}
```

## 致谢
目前看起来，不用登陆，还能免费下载和使用的ip库，只剩下db-ip了。