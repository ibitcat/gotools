/*
author：domicat
date： 2017-2-14
brief：xlsx转lua工具，支持id重复检测，支持json格式错误检测。
*/

package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/xuri/excelize"
)

const (
	E_NONE = iota
	E_WARN
	E_ERROR
)

func GetFullPath(path string) string {
	absolutePath, _ := filepath.Abs(path)
	return absolutePath
}

// 获取两个时间戳的毫秒差
func GetDurationMs(t time.Time) int {
	return int(time.Now().Sub(t).Nanoseconds() / 1e6)
}

func checkErr(e error) {
	if e != nil {
		panic(e)
	}
}

func checkJson(text string) error {
	var temp interface{}
	err := json.Unmarshal([]byte(text), &temp)
	if err == nil {
		switch temp.(type) {
		case map[string]interface{}:
		case []interface{}:
		default:
			err = errors.New("json格式错误")
		}
	}
	return err
}

// 获取文件名
func getFileName(file string) string {
	return strings.TrimSuffix(file, ".xlsx")
}

// 获取lua输出路径（文件夹，例如：/item/）
func genLuaDir(xlsxpath string, file string) string {
	p := strings.TrimSuffix(xlsxpath, file)
	p = strings.TrimPrefix(p, xlsxRoot)
	luaDir := luaRoot + p
	return luaDir
}

// 删除服务器无用的lua文件
func removeLua(xlsxpath string, file string) {
	fileName := getFileName(file)
	luaDir := genLuaDir(xlsxpath, file)
	luaFile := fmt.Sprintf("%s%s.lua", luaDir, fileName)
	os.Remove(luaFile)
}

func loadLang(xlsxpath string) ([][]string, map[string]int, map[string]int) {
	p := strings.TrimPrefix(xlsxpath, xlsxRoot)
	sep := "/"
	if runtime.GOOS == "windows" {
		sep = "\\"
	}
	p = strings.Replace(p, sep, "$", -1)
	langFile := langRoot + sep + p

	xlFile, err := excelize.OpenFile(langFile)
	if err != nil {
		return nil, nil, nil
	}

	sheet := xlFile.GetRows("Sheet1")
	FieldRef := make(map[string]int)
	IdRef := make(map[string]int)
	for i, row := range sheet {
		if i == 0 { //第一行
			for j, text := range row {
				if strings.Contains(text, "_翻译") {
					fieldName := strings.TrimRight(text, "_翻译")
					if _, ok := FieldRef[fieldName]; ok {
						FieldRef[fieldName] = j
					}
				} else {
					FieldRef[text] = j
				}
			}
		} else {
			IdRef[row[0]] = i
		}
	}

	return sheet, FieldRef, IdRef
}

// 解析配置头
func loadXlsxHead(workSheet [][]string) (ErrorInfo, map[int]FieldInfo) {
	fieldRow := workSheet[1] //字段名
	typeRow := workSheet[2]  //字段类型
	modeRow := workSheet[3]  //生成方式
	if len(fieldRow) == 0 {
		return ErrorInfo{E_ERROR, "字段名为空"}, nil
	}

	Field := make(map[int]FieldInfo, 50) //最多50个字段
	for i, fieldName := range fieldRow {
		fieldType := typeRow[i]
		modeType := modeRow[i]

		// 检测key字段
		if i == 0 { //字段行的第一个字段为配置的key,需要检查下
			if len(fieldName) == 0 {
				return ErrorInfo{E_ERROR, "配置没有key"}, nil
			}
			if fieldType != "int" && fieldType != "string" {
				return ErrorInfo{E_ERROR, "id字段类型错误"}, nil
			}
		}

		// 字段通用检查
		if len(fieldName) > 0 {
			if strings.Contains(fieldName, " ") {
				return ErrorInfo{E_ERROR, fmt.Sprintf("字段名[%s]有空格", fieldName)}, nil
			}
			if modeType != "c" && modeType != "s" && modeType != "d" {
				return ErrorInfo{E_ERROR, fmt.Sprintf("字段[%s]生成方式错误", fieldName)}, nil
			}
			if len(fieldType) == 0 {
				return ErrorInfo{E_ERROR, "字段类型不存在"}, nil
			}
			Field[i] = FieldInfo{fieldName, fieldType, modeType}
		} else {
			if len(modeType) > 0 || len(fieldType) > 0 {
				return ErrorInfo{E_ERROR, fmt.Sprintf("第%d个字段名为空", i+1)}, nil
			}
			break
		}
	}

	return ErrorInfo{E_NONE, "OK"}, Field
}

// 检查能否翻译
func checkTranslation(src, dst []string) bool {
	if len(src) != len(dst) {
		return false
	}

	for i, v := range src {
		if v != dst[i] {
			return false
		}
	}
	return true
}

// 解析成lua格式
func parseToLua(xlsxpath string, file string, workSheet [][]string, Field map[int]FieldInfo, errInfo *ErrorInfo) {
	var rowsSlice []string

	// 检查key字段
	keyField := Field[0]
	key := Field[0].Name
	checkOnly := false
	if keyField.Mode != "s" && keyField.Mode != "d" {
		removeLua(xlsxpath, file)
		errInfo.Level = E_WARN
		errInfo.ErrMsg = "不需要生成"

		checkOnly = true //客户端配置，仅需要检查json格式
	}
	if !checkOnly {
		sliceLen := len(workSheet) * (len(Field) + 2)
		rowsSlice = make([]string, 0, sliceLen)
		rowsSlice = append(rowsSlice, "--Don't Edit!!!") //第一行先占用

		// 配置
		rowsSlice = append(rowsSlice, "return {")
		rowsSlice = append(rowsSlice, "{")
	}

	// 配置是否需要翻译
	needTrans := false
	var fieldRef, idRef map[string]int
	var langSheet [][]string
	var re *regexp.Regexp
	if len(langRoot) > 0 {
		for _, f := range Field {
			if (f.Mode == "s" || f.Mode == "d") && f.Type == "string" {
				needTrans = true
				langSheet, fieldRef, idRef = loadLang(xlsxpath) // 读取翻译文件
				re = regexp.MustCompile(`%[a-z]`)               // 匹配占位符
				break
			}
		}
	}

	// 从配置的第4行开始解析
	idMap := make(map[string]bool) //检查id是否重复
forLable:
	for i := 4; i < len(workSheet); i++ {
		row := workSheet[i]
		if len(row) > 0 {
			var id string
			for idx, text := range row {
				// fields
				if f, ok := Field[idx]; ok {
					// key 字段
					if idx == 0 {
						id = text
						if len(text) == 0 { //如果key字段的值为空，则停止解析配置
							break forLable
						} else {
							if _, ok := idMap[text]; ok {
								errInfo.Level = E_ERROR
								errInfo.ErrMsg = fmt.Sprintf("id重复,第%d行,id=%s", i+1, text)
								return
							} else {
								idMap[text] = true
							}

							if !checkOnly {
								var str string
								if f.Type == "int" {
									str = fmt.Sprintf("    [%s] = {", text)
								} else {
									str = fmt.Sprintf("    ['%s'] = {", text)
								}

								rowsSlice = append(rowsSlice, str)
							}
						}
					}

					if len(text) <= 0 {
						continue
					}

					// json格式是否正确
					if f.Type == "table" {
						if needTrans {
							rId, rOk := idRef[id]
							cId, cOk := fieldRef[f.Name]
							if rOk && cOk && len(langSheet) > rId && len(langSheet[rId]) > cId {
								trCell := langSheet[rId][cId]
								if len(trCell) > 0 {
									err := checkJson(text)
									if err != nil {
										errInfo.Level = E_ERROR
										errInfo.ErrMsg = fmt.Sprintf("翻译json格式错误,第%d行,字段%s", rId+1, f.Name)
										return
									}
								}
							}
						} else {
							err := checkJson(text)
							if err != nil {
								errInfo.Level = E_ERROR
								errInfo.ErrMsg = fmt.Sprintf("json格式错误,第%d行,字段%s", i+1, f.Name)
								return
							}
						}
					}

					// 只生成服务器需要的字段
					if !checkOnly && (f.Mode == "s" || f.Mode == "d") {
						var str string
						if f.Type == "int" || f.Type == "number" {
							str = fmt.Sprintf("        ['%s'] = %s,", f.Name, text)
						} else {
							if f.Type == "table" {
								// 压缩成一行
								text = strings.Replace(text, " ", "", -1)
								text = strings.Replace(text, "\n", "", -1)
							} else if f.Type == "string" { // ' 替换成 \'
								// 翻译
								if needTrans {
									rId, rOk := idRef[id]
									cId, cOk := fieldRef[f.Name]
									if rOk && cOk && len(langSheet) > rId && len(langSheet[rId]) > cId {
										trCell := langSheet[rId][cId]
										if len(trCell) > 0 {
											checkOk := true
											if file == "string.xlsx" ||
												file == "error.xlsx" {
												reSlice1 := re.FindAllString(text, -1)
												reSlice2 := re.FindAllString(trCell, -1)
												if !checkTranslation(reSlice1, reSlice2) {
													errInfo.Level = E_ERROR
													errInfo.ErrMsg = fmt.Sprintf("翻译错误,id=%s,字段%s", id, f.Name)
													checkOk = false
												}
											}
											if checkOk {
												text = trCell
											}
										}
									}
								}
								text = strings.Replace(text, "'", `\'`, -1)
							}
							str = fmt.Sprintf("        ['%s'] = '%s',", f.Name, text)
						}
						rowsSlice = append(rowsSlice, str)
					}
				}
			}

			if !checkOnly {
				rowsSlice = append(rowsSlice, "    },")
			}
		}
	}

	// 配置解析完毕，添加字段表
	if !checkOnly {
		rowsSlice = append(rowsSlice, "},")
		// 字段table
		idxs := make([]int, 0, len(Field))
		for k, _ := range Field {
			idxs = append(idxs, k)
		}
		sort.Ints(idxs)
		rowsSlice = append(rowsSlice, "\n{")
		for _, v := range idxs {
			f := Field[v]
			if f.Mode == "s" || f.Mode == "d" {
				rowsSlice = append(rowsSlice, fmt.Sprintf("    ['%s'] = '%s',", f.Name, f.Type))
			}
		}
		rowsSlice = append(rowsSlice, "},\n")

		// key
		rowsSlice = append(rowsSlice, fmt.Sprintf("'%s'", key))
		rowsSlice = append(rowsSlice, "}")

		// 生成lua文件
		outPutToLua(xlsxpath, file, rowsSlice)
	}
}

// 目录不存在则新建目录
func outPutToLua(xlsxpath string, file string, rowsSlice []string) {
	fileName := getFileName(file)
	luaDir := genLuaDir(xlsxpath, file)

	_, err := os.Stat(luaDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(luaDir, os.ModePerm)
		if err != nil {
			//panic("mkdir failed!")
			return
		}
	}

	//md5检测
	/*
		luaStr := strings.Join(rowsSlice, "\n")
		md5h := md5.New()
		md5h.Write([]byte(luaStr))
		md5str := fmt.Sprintf("%x", md5h.Sum([]byte("")))
		if v, ok := luaMap[p+fileName]; ok && md5str == v {
			panic("内容没有变化")
		} else {
			rowsSlice[0] = fmt.Sprintf("--md5:%s", md5str)
		}
	*/

	luaFile := fmt.Sprintf("%s%s.lua", luaDir, fileName)
	outFile, operr := os.OpenFile(luaFile, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
	checkErr(operr)
	defer outFile.Close()

	outFile.WriteString(strings.Join(rowsSlice, "\n"))
	outFile.Sync()
}

func loadXlsx(xlsxpath string, file string) {
	errInfo := ErrorInfo{E_NONE, "OK"}
	durationms := 0
	defer func() {
		if err := recover(); err != nil {
			errInfo.ErrMsg = fmt.Sprintf("%v", err)
		}
		resChan <- Result{xlsxpath, errInfo, durationms}
	}()

	startTime := time.Now()
	xlFile, err := excelize.OpenFile(xlsxpath)
	if err != nil {
		errInfo.Level = E_ERROR
		errInfo.ErrMsg = err.Error()
		return
	}

	// 第一个sheet为配置
	workSheet := xlFile.GetRows("Sheet1")
	if len(workSheet) < 4 {
		errInfo.Level = E_ERROR
		errInfo.ErrMsg = "配置头至少要4行"
		return
	}

	eret, fields := loadXlsxHead(workSheet)
	if eret.Level == E_ERROR {
		errInfo.Level = E_ERROR
		errInfo.ErrMsg = eret.ErrMsg
		return
	}

	parseToLua(xlsxpath, file, workSheet, fields, &errInfo)
	durationms = GetDurationMs(startTime)
}

func walkXlsx(path string) {
	err := filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		ok, mErr := filepath.Match("[^~$]*.xlsx", f.Name())
		if ok {
			if f == nil {
				return err
			}
			if f.IsDir() {
				return nil
			}

			xlsxSlice = append(xlsxSlice, XlsxPath{path, f.Name(), uint64(f.ModTime().UnixNano() / 1000000)})
			return nil
		}
		return mErr
	})
	checkErr(err)
}

func loadLastModTime() {
	file, ferr := os.Open("lastModTime.txt")
	if ferr != nil {
		color.Red("注意：全部重新生成!!!")
		needForce = true
		os.RemoveAll(luaRoot)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 {
			s := strings.Split(line, "|")
			if len(s) == 2 {
				tm, _ := strconv.ParseUint(s[1], 10, 64)
				lastModTime[s[0]] = tm
			}
		}
	}
}

func saveLastModTime() {
	outFile, operr := os.OpenFile("lastModTime.txt", os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
	checkErr(operr)
	defer outFile.Close()

	modTimes := make([]string, 0, len(xlsxSlice))
	for _, xp := range xlsxSlice {
		modTimes = append(modTimes, xp.PathName+"|"+strconv.FormatUint(xp.ModTime, 10))
	}

	outFile.WriteString(strings.Join(modTimes, "\n"))
	outFile.Sync()
}

type ErrorInfo struct {
	Level  int //0=成功 1=警告 2=错误
	ErrMsg string
}

type Result struct {
	Name string
	ErrorInfo
	Msec int
}

type XlsxPath struct {
	PathName string
	FileName string
	ModTime  uint64
}

type FieldInfo struct {
	Name string
	Type string
	Mode string
}

var luaRoot string
var xlsxRoot string
var langRoot string
var resChan chan Result
var xlsxSlice []XlsxPath
var lastModTime map[string]uint64
var needForce bool = false

func main() {
	flag.StringVar(&xlsxRoot, "i", "", "输入路径")
	flag.StringVar(&luaRoot, "o", "", "输出路径")
	flag.StringVar(&langRoot, "l", "", "翻译文件路径")
	flag.Parse()
	if len(luaRoot) == 0 || len(xlsxRoot) == 0 {
		color.Red("输入路径或输出路径为空")
		return
	}

	xlsxSlice = make([]XlsxPath, 0, 100)
	startTime := time.Now()
	walkXlsx(xlsxRoot)

	// 最后modify的时间
	lastModTime = make(map[string]uint64, len(xlsxRoot))
	loadLastModTime()

	//f, err := os.Create("cpu-profile.prof")
	//if err != nil {
	//	fmt.Println(err)
	//}
	//pprof.StartCPUProfile(f)

	// 找到有变化的
	changedXlsx := make([]XlsxPath, 0, len(xlsxSlice))
	for _, xp := range xlsxSlice {
		lastModTm, ok := lastModTime[xp.PathName]
		if ok && lastModTm == xp.ModTime {
			//xlsxSlice = append(xlsxSlice[:i], xlsxSlice[i+1:]...)
			//fmt.Println(xp.PathName + "无变化")
		} else {
			if !needForce {
				color.Red("[有变化的文件]： " + xp.PathName)
			}
			changedXlsx = append(changedXlsx, xp)
		}
	}

	count := len(changedXlsx)
	if count > 0 {
		resChan = make(chan Result, count)
		fmt.Println("正在生成，请稍后……,条目=", count)
		for _, xp := range changedXlsx {
			go loadXlsx(xp.PathName, xp.FileName)
		}

		resSlice := make([]Result, 0, count)
		for i := 0; i < count; i++ {
			res := <-resChan
			resSlice = append(resSlice, res)
		}

		msec := GetDurationMs(startTime)
		printResult(resSlice, msec)
	} else {
		color.Green("无需生成^_^")
	}

	//pprof.StopCPUProfile()
}

func printResult(resSlice []Result, msec int) {
	fmt.Println("生成结束，结果如下：")

	sort.Slice(resSlice, func(i, j int) bool { return resSlice[i].Name < resSlice[j].Name })
	allOk := true
	for _, res := range resSlice {
		if res.Level == 1 { //警告
			color.Set(color.FgYellow)
			fmt.Printf("%-60s %-s\n", res.Name, res.ErrMsg)
			color.Unset()
		} else if res.Level == 2 { //错误
			if allOk {
				allOk = false
			}
			color.Set(color.FgRed)
			fmt.Printf("%-60s %-s\n", res.Name, res.ErrMsg)
			color.Unset()
		} else {
			fmt.Printf("%-60s %-s %5sms\n", res.Name, res.ErrMsg, strconv.Itoa(res.Msec))
		}
	}

	if allOk {
		fmt.Printf("生成完毕，条目=%d，耗时=%d 毫秒 ！\n", len(resSlice), msec)

		// 清除废弃的lua
		/*
			fmt.Printf("开始清除废弃的lua文件……\n\n")
			for _, lua := range luaSlice {
				p := strings.TrimSuffix(lua.PathName, "lua")
				p = strings.TrimPrefix(p, "l-")
				xlsxdir := p + "xlsx"
				found := false
				for _, xlxs := range xlsxSlice {
					if xlxs.PathName == xlsxdir {
						found = true
						break
					}
				}
				if !found {
					fmt.Printf("删除lua文件[%s]\n", lua.PathName)
					os.Remove(lua.PathName)
				}
			}
			fmt.Println("清除废弃的lua文件完毕！")
		*/
		saveLastModTime()
	} else {
		color.Set(color.FgRed)
		fmt.Printf("生成有错误，条目=%d，耗时=%d 毫秒 ！\n", len(resSlice), msec)
		color.Unset()
	}
}
