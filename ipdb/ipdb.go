package ipdb

import (
	"compress/gzip"
	"errors"
	"github.com/oschwald/maxminddb-golang"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var (
	downloadLock sync.Mutex        // 下载锁
	ipDb         *maxminddb.Reader // 数据库
)

// Get 获取实例
func Get() *maxminddb.Reader {
	return ipDb
}

// 获取目录内的版本
func getLastDatabaseFileName() string {
	exeDir := filepath.Dir(os.Args[0])
	fs, _ := ioutil.ReadDir(exeDir)
	dbReg := regexp.MustCompile(`dbip-city-lite-\d{4}-\d{2}.mmdb`)
	for _, file := range fs {
		if !file.IsDir() {
			if dbReg.MatchString(file.Name()) {
				//log.Println(exeDir + file.Name())
				return file.Name()
			}
		}
	}
	return ""
}

// downloadDatabase 下载并且解压
func downloadDatabase(url string) (string, error) {
	filename := filepath.Base(url)
	if !strings.HasSuffix(filename, ".gz") {
		return "", errors.New("not gz file")
	}
	filename = filename[:len(filename)-3]

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 解压
	g, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", err
	}

	absFile := filepath.Join(filepath.Dir(os.Args[0]), filename)
	// Create the file
	out, err := os.Create(absFile)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, g)
	return filename, err
}

func openIPDb(dbFile string) error {
	if dbFile == "" {
		return nil
	}

	if ipDb != nil {
		ipDb.Close()
		ipDb = nil
	}

	newIpDb, err := maxminddb.Open(filepath.Join(filepath.Dir(os.Args[0]), dbFile))
	if err != nil {
		log.Println("[WARNING] open ip db failed:", err)
		return err
	}

	ipDb = newIpDb
	return nil
}

// UpdateIpDatabase 更新ip数据库，如果没有就下载最新的版本
func UpdateIpDatabase() {
	downloadLock.Lock()
	defer downloadLock.Unlock()

	lastDatabaseFileName := getLastDatabaseFileName()
	if len(lastDatabaseFileName) > 0 {
		openIPDb(lastDatabaseFileName)
	}

	// 解析下载地址
	resp, err := http.Get("https://db-ip.com/db/download/ip-to-city-lite")
	if err != nil {
		log.Println("[WARNING] fetch ip city lite database failed:", err)
		return
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("[WARNING] fetch html failed:", err)
		return
	}
	//https://download.db-ip.com/free/dbip-city-lite-2022-07.mmdb.gz
	urlRegex := regexp.MustCompile(`<a href='(.*?\.gz)' class='.*?'>Download IP to City Lite MMDB</a>`)
	urls := urlRegex.FindAllStringSubmatch(string(b), 1)
	if len(urls) == 0 {
		log.Println("[WARNING] fetch download url failed:", err)
		return
	}
	downloadUrl := urls[0][1]

	// 是否已经是最新
	if lastDatabaseFileName+".gz" == filepath.Base(downloadUrl) {
		log.Println("ip database is the latest")
		return
	}

	// 下载数据库
	lastIPDatabase, err := downloadDatabase(downloadUrl)
	if err != nil {
		log.Println("[WARNING] download ip city lite database failed:", err)
		return
	}

	openIPDb(lastIPDatabase)

	// 删除历史文件
	if len(lastDatabaseFileName) > 0 {
		os.Remove(lastDatabaseFileName)
	}
}
