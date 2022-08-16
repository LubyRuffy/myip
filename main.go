package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/LubyRuffy/myip/ipdb"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/acme/autocert"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	version          = "v0.2" // 版本，展示用
	processedRequest uint64   // 处理了多少请求，统计用
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do stuff here
		atomic.AddUint64(&processedRequest, 1)

		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		log.Println(ip, r.RequestURI)

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

// getIp 获取ip， 返回第一个是连接服务器的ip，第二个是upstream的ip
func getIp(r *http.Request) (string, string) {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)

	var upstream string
	if r.Header.Get("X-Forwarded-For") != "" {
		upstream = r.Header.Get("X-Forwarded-For")
		if strings.Contains(upstream, ",") {
			// 多条，取第一条
			upstream = strings.Split(upstream, ",")[0]
		}
	} else if r.Header.Get("X-Real-IP") != "" {
		upstream = r.Header.Get("X-Real-IP")
	} else {
		upstream = ip
	}

	return ip, upstream
}

// headerString http request 转换为字符串
func headerString(r *http.Request) string {
	var s string
	s = fmt.Sprintf("%s %s %s\n", r.Method, r.RequestURI, r.Proto)
	for k, v := range r.Header {
		s += k + ": " + strings.Join(v, ",") + "\n"
	}
	return s
}

func defaultAction(w http.ResponseWriter, r *http.Request) {
	ip, _ := getIp(r)
	w.Write([]byte(ip))
}

/*
curl 127.0.0.1:8888/ip
{"ip":"127.0.0.1","real_ip":"127.0.0.1"}
curl 127.0.0.1:8888/ip -H 'X-Forwarded-For: 1.1.1.1'
{"ip":"127.0.0.1","real_ip":"1.1.1.1"}
curl 127.0.0.1:8888/ip -H 'X-Real-IP: 2.2.2.2'
{"ip":"127.0.0.1","real_ip":"2.2.2.2"}
curl 127.0.0.1:8888/ip -H 'X-Forwarded-For: 1.1.1.1' -H 'X-Real-IP: 2.2.2.2'
{"ip":"127.0.0.1","real_ip":"2.2.2.2"}
*/
func ipAction(w http.ResponseWriter, r *http.Request) {
	ip, upstream := getIp(r)

	result := map[string]interface{}{
		"ip": ip,
	}

	if upstream != ip {
		result["upstream"] = upstream
	}

	prettyJsonOutput(w, r, result)
}

/*
curl 127.0.0.1:8888/h
curl 127.0.0.1:8888/header
{"header":"GET /header HTTP/1.1\nUser-Agent: curl/7.83.0\nAccept: *\n","ip":"127.0.0.1","upstream":"127.0.0.1"}
curl 127.0.0.1:8888/header -H 'X-Forwarded-For: 8.8.8.8'
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
func headerAction(w http.ResponseWriter, r *http.Request) {
	ip, upstream := getIp(r)

	result := map[string]interface{}{
		"ip":     ip,
		"header": headerString(r),
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

	prettyJsonOutput(w, r, result)
}

func prettyJsonOutput(w http.ResponseWriter, r *http.Request, result interface{}) {
	enc := json.NewEncoder(w)
	if r.URL.Query().Has("p") || r.URL.Query().Has("pretty") {
		enc.SetIndent("", "  ")
	}

	enc.Encode(result)
}

/*
curl 127.0.0.1:8888/geo -H 'X-Forwarded-For: 8.8.8.8'
127.0.0.1
8.8.8.8,United States,California,Mountain View,North America
*/
func geoAction(w http.ResponseWriter, r *http.Request) {
	ip, upstream := getIp(r)

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

	w.Write([]byte(ipLine))
}

// curl 127.0.0.1:8888/c -H 'X-Forwarded-For: 8.8.8.8'
// {"ip":"127.0.0.1","upstream":"8.8.8.8","upstream_country":"United States"}
func countryAction(w http.ResponseWriter, r *http.Request) {
	ip, upstream := getIp(r)

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

	prettyJsonOutput(w, r, result)
}

func statusAction(w http.ResponseWriter, r *http.Request) {
	result := map[string]interface{}{
		"status":  "ok",
		"version": version,
	}
	if ipdb.Get() != nil {
		result["ipdb"] = ipdb.Get().Metadata
	}
	prettyJsonOutput(w, r, result)
}

func runWeb(addr string, subdomain []string) []*http.Server {
	router := mux.NewRouter()
	router.Use(loggingMiddleware) //统计和日志
	// 首页
	router.HandleFunc("/", defaultAction)
	// ip
	router.HandleFunc("/ip", ipAction)
	// geo
	router.HandleFunc("/geo", geoAction)
	router.HandleFunc("/g", geoAction)
	// header
	router.HandleFunc("/h", headerAction)
	router.HandleFunc("/header", headerAction)
	// country
	router.HandleFunc("/c", countryAction)
	router.HandleFunc("/country", countryAction)
	// status
	router.HandleFunc("/status", statusAction)

	if len(subdomain) > 0 {
		m := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(subdomain...),
		}

		s := &http.Server{
			Addr:      ":443",
			TLSConfig: m.TLSConfig(),
			Handler:   router,
		}

		//redirect := func(w http.ResponseWriter, req *http.Request) {
		//	target := "https://" + req.Host + req.URL.Path
		//	if len(req.URL.RawQuery) > 0 {
		//		target += "?" + req.URL.RawQuery
		//	}
		//	http.Redirect(w, req, target, http.StatusTemporaryRedirect)
		//}

		httpSrv := &http.Server{
			Addr:      addr,
			TLSConfig: m.TLSConfig(),
			Handler:   router,
			//Handler:   m.HTTPHandler(http.HandlerFunc(redirect)),
		}

		go httpSrv.ListenAndServe() // http用于验证

		go s.ListenAndServeTLS("", "")

		return []*http.Server{httpSrv, s}
	}

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go srv.ListenAndServe()
	return []*http.Server{srv}
}

func main() {
	updateDuration := time.Hour * 24 // 1天检查一次更新

	addr := flag.String("addr", ":80", "listen addr")
	duration := flag.Int("duration", 10, "duration")
	autotls := flag.Bool("autotls", false, "let's encrypt")
	subdomains := flag.String("subdomains", "", "only useful when autotls enable")
	flag.Parse()

	// 检查数据库
	go ipdb.UpdateIpDatabase()

	var subdomain []string
	if *autotls {
		subdomain = strings.Split(*subdomains, ",")
		if len(subdomain) == 0 {
			panic("subdomains cannot be empty when autotls enable")
		}
	}
	srvs := runWeb(*addr, subdomain)

	for _, s := range srvs {
		log.Println("listen at:", s.Addr)
	}

	// 等待事件
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ticker := time.NewTicker(time.Duration(*duration) * time.Second)

	lastCheckTime := time.Now()
	for {
		select {
		case <-ticker.C:
			//
			if time.Since(lastCheckTime) > updateDuration {
				// 检查数据库
				go ipdb.UpdateIpDatabase()
			}
			log.Println("=== processed:", processedRequest)
		case <-sigs:
			for _, s := range srvs {
				s.Close()
			}

			ticker.Stop()
			return
		}
	}
}
