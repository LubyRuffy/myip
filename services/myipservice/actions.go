package myipservice

import (
	"fmt"
	"github.com/LubyRuffy/myip/ipdb"
	"github.com/gin-gonic/gin"
	"net"
	"net/http"
	"strings"
)

// getIp 获取ip， 返回第一个是连接服务器的ip，第二个是upstream的ip
func getIp(c *gin.Context) (string, string) {
	ip, _, _ := net.SplitHostPort(c.Request.RemoteAddr)

	var upstream string
	if c.Request.Header.Get("X-Forwarded-For") != "" {
		upstream = c.Request.Header.Get("X-Forwarded-For")
		if strings.Contains(upstream, ",") {
			// 多条，取第一条
			upstream = strings.Split(upstream, ",")[0]
		}
	} else if c.Request.Header.Get("X-Real-IP") != "" {
		upstream = c.Request.Header.Get("X-Real-IP")
	} else {
		upstream = ip
	}

	return ip, upstream
}

// headerString http request 转换为字符串
func headerString(c *gin.Context) string {
	var s string
	s = fmt.Sprintf("%s %s %s\n", c.Request.Method, c.Request.RequestURI, c.Request.Proto)
	for k, v := range c.Request.Header {
		s += k + ": " + strings.Join(v, ",") + "\n"
	}
	return s
}

func defaultAction(c *gin.Context) {
	ip, _ := getIp(c)
	c.String(http.StatusOK, ip)
}

/*
curl 127.0.0.1/ip
{"ip":"127.0.0.1","real_ip":"127.0.0.1"}
curl 127.0.0.1/ip -H 'X-Forwarded-For: 1.1.1.1'
{"ip":"127.0.0.1","real_ip":"1.1.1.1"}
curl 127.0.0.1/ip -H 'X-Real-IP: 2.2.2.2'
{"ip":"127.0.0.1","real_ip":"2.2.2.2"}
curl 127.0.0.1/ip -H 'X-Forwarded-For: 1.1.1.1' -H 'X-Real-IP: 2.2.2.2'
{"ip":"127.0.0.1","real_ip":"2.2.2.2"}
*/
func ipAction(c *gin.Context) {
	ip, upstream := getIp(c)

	result := map[string]interface{}{
		"ip": ip,
	}

	if upstream != ip {
		result["upstream"] = upstream
	}

	PrettyJsonOutput(c, result)
}

/*
curl 127.0.0.1/h
curl 127.0.0.1/header
{"header":"GET /header HTTP/1.1\nUser-Agent: curl/7.83.0\nAccept: *\n","ip":"127.0.0.1","upstream":"127.0.0.1"}
curl 127.0.0.1/header -H 'X-Forwarded-For: 8.8.8.8'
{
  "geo": {
    "city": {
      "names": {
        "en": "Mountain View"
      }
    },
    "continent": {
      "code": "NA",
      "geoname_id": 6255149,
      "names": {
        "de": "Nordamerika",
        "en": "North America",
        "es": "Norteamérica",
        "fa": " امریکای شمالی",
        "fr": "Amérique Du Nord",
        "ja": "北アメリカ大陸",
        "ko": "북아메리카",
        "pt-BR": "América Do Norte",
        "ru": "Северная Америка",
        "zh-CN": "北美洲"
      }
    },
    "country": {
      "geoname_id": 6252001,
      "is_in_european_union": false,
      "iso_code": "US",
      "names": {
        "de": "Vereinigte Staaten von Amerika",
        "en": "United States",
        "es": "Estados Unidos de América (los)",
        "fa": "ایالات متحدهٔ امریکا",
        "fr": "États-Unis",
        "ja": "アメリカ合衆国",
        "ko": "미국",
        "pt-BR": "Estados Unidos",
        "ru": "США",
        "zh-CN": "美国"
      }
    },
    "location": {
      "latitude": 37.4223,
      "longitude": -122.085
    },
    "subdivisions": [
      {
        "names": {
          "en": "California"
        }
      }
    ]
  },
  "header": "GET /header HTTP/1.1\nUser-Agent: curl/7.83.0\nAccept: *\nX-Forwarded-For: 8.8.8.8\n",
  "ip": "127.0.0.1",
  "upstream": "8.8.8.8"
}
*/
func headerAction(c *gin.Context) {
	ip, upstream := getIp(c)

	result := map[string]interface{}{
		"ip":     ip,
		"header": headerString(c),
	}

	if upstream != ip {
		result["upstream"] = upstream
	}

	if ipdb.Get() != nil {
		city, err := ipdb.Get().City(net.ParseIP(ip))
		if err == nil && city != nil {
			result["geo"] = city
		}

		if upstream != ip {
			city1, err := ipdb.Get().City(net.ParseIP(upstream))
			if err == nil && city1 != nil {
				result["upstream_geo"] = city1
			}
		}
	}

	PrettyJsonOutput(c, result)
}

// PrettyJsonOutput 决定是否格式化json输出
func PrettyJsonOutput(c *gin.Context, result interface{}) {
	if c.Request.URL.Query().Has("p") || c.Request.URL.Query().Has("pretty") {
		c.IndentedJSON(http.StatusOK, result)
	} else {
		c.JSON(http.StatusOK, result)
	}
}

/*
curl 127.0.0.1/geo -H 'X-Forwarded-For: 8.8.8.8'
127.0.0.1
8.8.8.8,United States,California,Mountain View,North America
*/
func geoAction(c *gin.Context) {
	ip, upstream := getIp(c)

	ipLine := ip
	var upstreamLine string
	if upstream != ip {
		upstreamLine = upstream
	}

	if ipdb.Get() != nil {
		city, err := ipdb.Get().City(net.ParseIP(ip))
		if err == nil && city != nil {
			ipLine += "," + city.Country.Names["en"]

			var subdivisions string
			if len(city.Subdivisions) > 0 {
				subdivisions = city.Subdivisions[0].Names["en"]
			}
			ipLine += "," + subdivisions
			ipLine += "," + city.City.Names["en"]
			ipLine += "," + city.Continent.Names["en"]
		}

		if upstream != ip {
			city, err = ipdb.Get().City(net.ParseIP(upstream))
			if err == nil && city != nil {
				upstreamLine += "," + city.Country.Names["en"]
				var subdivisions string
				if len(city.Subdivisions) > 0 {
					subdivisions = city.Subdivisions[0].Names["en"]
				}
				upstreamLine += "," + subdivisions
				upstreamLine += "," + city.City.Names["en"]
				upstreamLine += "," + city.Continent.Names["en"]
			}
		}

	}

	if len(upstreamLine) > 0 {
		ipLine += "\n" + upstreamLine
	}

	c.String(http.StatusOK, ipLine)
}

// curl 127.0.0.1/c -H 'X-Forwarded-For: 8.8.8.8'
// {"ip":"127.0.0.1","upstream":"8.8.8.8","upstream_country":"United States"}
func countryAction(c *gin.Context) {
	ip, upstream := getIp(c)

	result := map[string]interface{}{
		"ip": ip,
	}

	if upstream != ip {
		result["upstream"] = upstream
	}

	if ipdb.Get() != nil {
		city, err := ipdb.Get().City(net.ParseIP(ip))
		if err == nil && city != nil {
			result["country"] = city.Country.Names["en"]
		}

		if upstream != ip {
			city, err = ipdb.Get().City(net.ParseIP(upstream))
			if err == nil && city != nil {
				result["upstream_country"] = city.Country.Names["en"]
			}
		}
	}

	PrettyJsonOutput(c, result)
}

// RegisterActions 注册服务
func RegisterActions(router *gin.Engine) {
	// 首页
	router.Any("/", defaultAction)
	// ip
	router.Any("/ip", ipAction)
	// geo
	router.Any("/geo", geoAction)
	router.Any("/g", geoAction)
	// header
	router.Any("/h", headerAction)
	router.Any("/header", headerAction)
	// country
	router.Any("/c", countryAction)
	router.Any("/country", countryAction)
}
