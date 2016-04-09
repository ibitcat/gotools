package main

import (
	"container/list"
	"crypto/md5"
	"encoding/csv"
	"fmt"
	"github.com/go-fsnotify/fsnotify"
	"io"
	"io/ioutil"
	"log"
	"net/smtp"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// 文件列表
var locker sync.Mutex
var outputFileName string = "filesName.csv"
var md5Map map[string]string
var changeMap map[int]*list.List
var watcherMap map[string]*fsnotify.Watcher

var PrefixMap = map[int]string{
	1: "修改",
	2: "新建",
	3: "删除",
	4: "重命名",
}

var FileType = map[bool]string{
	true:  "文件夹",
	false: "文件",
}

// 错误检查
func CheckErr(err error) {
	if nil != err {
		panic(err)
	}
}

// 获取绝对路径
func GetFullPath(path string) string {
	absolutePath, _ := filepath.Abs(path)
	return absolutePath
}

// 输出文件是否存在
func OutputFileIsExist() bool {
	_, err := os.Stat(outputFileName)
	return err == nil || os.IsExist(err)
}

// 初始化
func Init() {
	if OutputFileIsExist() {
		file, err := os.Open(outputFileName) // 读取csv文件
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer file.Close()

		reader := csv.NewReader(file)
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				fmt.Println("Error:", err)
				return
			}

			if len(record) > 0 {
				md5Map[record[0]] = record[1]
			}
		}
	}
}

// 生成文件的md5
func GenMd5(fileName string) string {
	file, err := os.OpenFile(fileName, os.O_RDONLY, 0644)
	defer file.Close()

	Buf, buferr := ioutil.ReadFile(fileName)
	if buferr == nil {
		fmt.Sprintf("%x", md5.Sum(Buf))
	}

	if err == nil {
		md5h := md5.New()
		io.Copy(md5h, file)
		return fmt.Sprintf("%x", md5h.Sum([]byte("")))
	} else {
		fmt.Println("gem md5 fail", fileName)
	}

	return ""
}

// 压入文件名
func PrintFilesName(path string) error {
	fullPath := GetFullPath(path)

	fmt.Println("顶层绝对路径 path = ", fullPath)

	return filepath.Walk(fullPath, func(pathName string, fi os.FileInfo, err error) error {
		if nil == fi {
			return err
		}

		//log.Println("filepath.Walk=", pathName)
		if fi.IsDir() { // 文件夹
			MakeWatcher(pathName)
			if md5Str := GenMd5(pathName); md5Str != "" {
				md5Map[pathName] = md5Str
			}
		} else { // 文件
			name := fi.Name()
			if outputFileName != name {
				if md5Str := GenMd5(pathName); md5Str != "" {
					md5Map[pathName] = md5Str
				}
			}
		}

		return nil
	})
}

// 输出到csv文件
func OutputFilesName() {
	f, err := os.Create(outputFileName)
	//f, err := os.OpenFile(outputFileName, os.O_APPEND | os.O_CREATE, os.ModeAppend)
	CheckErr(err)
	defer f.Close()

	f.WriteString("\xEF\xBB\xBF") // 写入utf8-bom
	writer := csv.NewWriter(f)

	for k, v := range md5Map {
		isFolder := false
		fileInfo, err := os.Stat(k)
		if err == nil && fileInfo.IsDir() {
			isFolder = true
		}
		writer.Write([]string{FileType[isFolder], k, v})
	}

	writer.Write([]string{""})
	writer.Write([]string{"文件变更历史："})
	writer.Flush()
}

// 插入文件变更历史
func AppendChangeLog(typeStr string, fileName string, isFolder bool) {
	f, err := os.OpenFile(outputFileName, os.O_APPEND, os.ModeAppend)
	CheckErr(err)
	defer f.Close()

	writer := csv.NewWriter(f)

	writer.Write([]string{typeStr, FileType[isFolder], fileName, time.Now().String()})

	writer.Flush()
}

func InsertChangeMap(typeId int, fileName string) {
	locker.Lock()
	if _, ok := changeMap[typeId]; !ok {
		changeMap[typeId] = list.New()
	}

	_, ok := watcherMap[fileName]
	changeMap[typeId].PushBack(fileName)
	AppendChangeLog(PrefixMap[typeId], fileName, ok)
	locker.Unlock()
}

func OnTimer() {
	timer := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-timer.C:
			log.Println("onTimer……")
			locker.Lock()

			mailContent := ""
			for k, v := range changeMap {
				mailContent += PrefixMap[k] + "列表：<br>"
				for el := v.Front(); nil != el; el = el.Next() {
					mailContent += el.Value.(string) + "<br>"
				}

				mailContent += "<br>"
			}

			if len(mailContent) > 0 {
				changeMap = make(map[int]*list.List)
				//go SendMail(mailContent)
			}

			locker.Unlock()
		}
	}
}

// 发送邮件
func SendMail(mailContent string) error {
	user := "email@163.com"
	password := "password"
	host := "smtp.163.com:25"
	to := "toemail@qq.com"

	subject := "文件变更通知"

	body := `
    <html>
    <body>
    <h3>
    文件列表
    </h3>
	<br>
	<h5>
	%s
	</h5>
    </body>
    </html>
    `

	body = fmt.Sprintf(body, mailContent)

	//fmt.Println("send email", body)

	hp := strings.Split(host, ":")
	auth := smtp.PlainAuth("", user, password, hp[0])
	content_type := "Content-Type: text/" + "html" + "; charset=UTF-8"

	msg := []byte("To: " + to + "\r\nFrom: " + user + "<" + user + ">\r\nSubject: " + subject + "\r\n" + content_type + "\r\n\r\n" + body)
	send_to := strings.Split(to, ";")
	err := smtp.SendMail(host, auth, user, send_to, msg)

	if err != nil {
		fmt.Println("send mail error!")
		fmt.Println(err)
	} else {
		fmt.Println("send mail success!")
	}
	return err
}

func WatcherGoroutine(watcher *fsnotify.Watcher) {
	for {
		select {
		case event := <-watcher.Events:
			//log.Println("event:", event)
			if event.Op&fsnotify.Write == fsnotify.Write { //修改文件
				oldMd5, ok := md5Map[event.Name]

				if ok {
					//log.Println("&&&&&&&&", event.Name)
					newMd5Str := GenMd5(event.Name)
					if newMd5Str != oldMd5 {
						md5Map[event.Name] = newMd5Str
						InsertChangeMap(1, event.Name)
					}
				} else {
					//log.Println("------", event.Name)
					md5Map[event.Name] = GenMd5(event.Name)
					InsertChangeMap(1, event.Name)
				}

			} else if event.Op&fsnotify.Create == fsnotify.Create { //新建文件
				// 添加新的watcher
				fileInfo, err := os.Stat(event.Name)
				//log.Println("新建的", event.Name, md5Map[event.Name])
				if err == nil && fileInfo.IsDir() {
					MakeWatcher(event.Name)
				}

				md5Map[event.Name] = GenMd5(event.Name)
				InsertChangeMap(2, event.Name)
			} else if event.Op&fsnotify.Remove == fsnotify.Remove { //文件移除
				if _, ok := md5Map[event.Name]; ok {
					delete(md5Map, event.Name)
					InsertChangeMap(3, event.Name)
				}

				// 删除watcher,注意，文件夹下不能有与文件夹同名的文件存在，必须要带后缀名
				DeleteWatcher(event.Name)
			} else if event.Op&fsnotify.Rename == fsnotify.Rename { //文件重命名
				if _, ok := md5Map[event.Name]; ok {
					delete(md5Map, event.Name)
					InsertChangeMap(4, event.Name)

					// 删除watcher,注意，文件夹下不能有与文件夹同名的文件存在，必须要带后缀名
					DeleteWatcher(event.Name)
				}
			}
		case err := <-watcher.Errors:
			log.Println("error:", err)
			return
		}
	}
}

func MakeWatcher(pathName string) {
	if _, ok := watcherMap[pathName]; !ok {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal("创建watcher失败……", pathName, err)
			watcher.Close()
			return
		}

		watcherMap[pathName] = watcher
	}

	//log.Println("add watcher……", pathName)
	go WatcherGoroutine(watcherMap[pathName])
	err := watcherMap[pathName].Add(pathName)
	if err != nil {
		log.Fatal(err)
	}
}

func DeleteWatcher(pathName string) {
	//log.Println("delete watcher--------", pathName)
	if watcher, ok := watcherMap[pathName]; ok {
		watcher.Remove(pathName)
		watcher.Close()
		delete(watcherMap, pathName)
	}
}

// main
func main() {
	//	var path string
	//	if len(os.Args) > 1 {
	//		path = os.Args[1]
	//	} else {
	//		path, _ = os.Getwd()
	//	}
	//log.Println(os.Getwd())

	// var
	md5Map = make(map[string]string)
	watcherMap = make(map[string]*fsnotify.Watcher)
	desPath := "D:/Code Monkey/Gopath/src/test"
	changeMap = make(map[int]*list.List)

	// 遍历文件夹
	err := PrintFilesName(desPath)
	if err != nil {
		log.Println(err)
		return
	}

	// 重新load所有文件的MD5，并输出到csv文件中（包括子目录）
	OutputFilesName()
	fmt.Println("load scv file done!")

	// 开启定时器
	go OnTimer()

	// 响应ctrl+c
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, os.Kill)
	<-done

	for k, v := range watcherMap {
		v.Remove(k)
		v.Close()
	}

	watcherMap = nil
	time.Sleep(1 * time.Second)

	fmt.Println("close wather!")
}
