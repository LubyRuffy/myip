package main

import (
	"flag"
	"github.com/LubyRuffy/myip/ipdb"
	"github.com/LubyRuffy/myip/services/myipservice"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/acme/autocert"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	version          = "v0.4" // 版本，展示用
	processedRequest uint64   // 处理了多少请求，统计用
)

func loggingMiddleware(c *gin.Context) {
	atomic.AddUint64(&processedRequest, 1)
}

func statusAction(c *gin.Context) {
	result := map[string]interface{}{
		"status":  "ok",
		"version": version,
	}
	if ipdb.Get() != nil {
		result["ipdb"] = myipservice.MarshalJSONWithTag(ipdb.Get().Metadata(), "maxminddb")
	}
	myipservice.PrettyJsonOutput(c, result)
}

func runWeb(addr string, subdomain []string) []*http.Server {
	router := gin.Default()
	router.Use(loggingMiddleware)       //统计和日志
	router.Any("/status", statusAction) // status

	myipservice.RegisterActions(router)

	var svrs []*http.Server
	if len(subdomain) > 0 {
		m := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(subdomain...),
			Cache:      autocert.DirCache("/tmp/autocert"), // 保存证书文件，复用
		}

		s := &http.Server{
			Addr:      ":443",
			TLSConfig: m.TLSConfig(),
			Handler:   router,
		}

		go s.ListenAndServeTLS("", "")

		svrs = append(svrs, s)
	}

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}
	go srv.ListenAndServe()

	svrs = append(svrs, srv)
	return svrs
}

func main() {
	updateDuration := time.Hour * 24 // 1天检查一次更新

	addr := flag.String("addr", ":80", "listen addr")
	duration := flag.Int("duration", 10, "duration")
	autotls := flag.Bool("autotls", false, "let's encrypt")
	subdomains := flag.String("subdomains", "", "only useful when autotls enable")
	downloadURL := flag.String("downloadIpDb", "", "only download dbfile") // https://download.db-ip.com/free/dbip-city-lite-2022-09.mmdb.gz
	flag.Parse()

	if *downloadURL != "" {
		if err := ipdb.UpdateIpDatabase(*downloadURL); err != nil {
			panic(err)
		}
		return
	}

	// 检查数据库
	go ipdb.UpdateIpDatabase("")

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
				go ipdb.UpdateIpDatabase("")
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
