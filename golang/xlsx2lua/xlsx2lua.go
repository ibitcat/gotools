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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/fatih/color"
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

func IsChineseChar(str string) bool {
	for _, r := range str {
		if unicode.Is(unicode.Scripts["Han"], r) {
			return true
		}
	}
	return false
}

func checkAscii(srcStr, desStr string) bool {
	srcBytes := make([]byte, 0, len(srcStr))
	for _, r := range srcStr {
		if r > 0x20 && r <= 0x7f && r != 0x2C && r != 0x2e { //忽略中英文逗号、句号的区别
			srcBytes = append(srcBytes, byte(r))
		}
	}
	byteLen := len(srcBytes)
	if byteLen > 0 {
		var idx int = 0
		for _, r := range desStr {
			if r > 0x20 && r <= 0x7f && byte(r) == srcBytes[idx] {
				idx++
				if idx >= byteLen {
					break
				}
			}
		}
		if byteLen != idx {
			//fmt.Println(byteLen, idx, string(srcBytes))
			return false
		}
	}
	return true
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

// 检查能否翻译
func checkTranslation(src, dst []string) (bool, string) {
	if len(src) != len(dst) {
		return false, "占位符个数不匹配"
	}

	for i, v := range src {
		if v != dst[i] {
			return false, fmt.Sprintf("第%d个占位符不匹配", i+1)
		}
	}
	return true, ""
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

// 读取翻译xlsx文件
func loadLangXlsx(xlsxpath string) *LangSheet {
	p := strings.TrimPrefix(xlsxpath, xlsxRoot)
	sep := "/"
	if runtime.GOOS == "windows" {
		sep = "\\"
	}
	p = strings.Replace(p, sep, "$", -1)
	langFile := langRoot + sep + p

	xlFile, err := excelize.OpenFile(langFile)
	if err != nil {
		return nil
	}

	//sheetName := strings.TrimSuffix(p, ".xlsx")
	sheetName := xlFile.GetSheetName(1)
	sheet := xlFile.GetRows(sheetName)
	langSheet := &LangSheet{
		SheetRows: sheet,
		FieldRef:  make(map[string]int),
		IdRef:     make(map[string]int),
	}

	for i, row := range sheet {
		if i == 0 { //第一行
			for j, text := range row {
				if strings.Contains(text, "_翻译") {
					fieldName := strings.TrimRight(text, "_翻译")
					langSheet.FieldRef[fieldName] = j
				}
			}
		} else {
			langSheet.IdRef[row[0]] = i
		}
	}

	return langSheet
}

func getLangCell(langSheet *LangSheet, id, field string) string {
	if langSheet != nil {
		rId, rOk := langSheet.IdRef[id]
		cId, cOk := langSheet.FieldRef[field]
		if rOk && cOk {
			return langSheet.SheetRows[rId][cId]
		}
	}
	return ""
}

// 解析配置头
func loadXlsxHead(workSheet [][]string) (ErrorInfo, map[int]FieldInfo) {
	fieldRow := workSheet[1] //字段名
	typeRow := workSheet[2]  //字段类型
	modeRow := workSheet[3]  //生成方式
	if len(fieldRow) == 0 {
		return ErrorInfo{E_ERROR, "[配置头错误]:配置字段名为空"}, nil
	}

	Field := make(map[int]FieldInfo, 50)
	for i, fieldName := range fieldRow {
		fieldType := typeRow[i]
		modeType := modeRow[i]

		// 检测key字段
		if i == 0 { //字段行的第一个字段为配置的key,需要检查下
			if len(fieldName) == 0 {
				return ErrorInfo{E_ERROR, "[配置头错误]:配置没有key"}, nil
			}
			if fieldType != "int" && fieldType != "string" {
				return ErrorInfo{E_ERROR, "[配置头错误]:id字段类型错误"}, nil
			}
		}

		// 字段通用检查
		if modeType != "r" {
			if len(fieldName) > 0 {
				if strings.Contains(fieldName, " ") {
					return ErrorInfo{E_ERROR, fmt.Sprintf("[配置头错误]:字段[%s]有空格", fieldName)}, nil
				}
				if modeType != "c" && modeType != "s" && modeType != "d" {
					return ErrorInfo{E_ERROR, fmt.Sprintf("[配置头错误]:字段[%s]生成方式错误", fieldName)}, nil
				}
				if len(fieldType) == 0 {
					return ErrorInfo{E_ERROR, fmt.Sprintf("[配置头错误]:字段[%s]类型不存在", fieldName)}, nil
				}
				Field[i] = FieldInfo{fieldName, fieldType, modeType}
			} else {
				if len(modeType) > 0 || len(fieldType) > 0 {
					return ErrorInfo{E_ERROR, fmt.Sprintf("[配置头错误]:第[%d]个字段名为空", i+1)}, nil
				}
				break
			}
		}
	}

	return ErrorInfo{E_NONE, "OK"}, Field
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

	// 检查翻译文件
	needTrans := false
	var langSheet *LangSheet
	var re *regexp.Regexp = nil
	if len(langRoot) > 0 {
		for _, f := range Field {
			if f.Mode == "s" || f.Mode == "d" {
				needTrans = true                   // 配置是否需要翻译
				langSheet = loadLangXlsx(xlsxpath) // 读取翻译文件
				if langSheet != nil && len(langSheet.SheetRows) == 0 {
					errInfo.Level = E_ERROR
					errInfo.ErrMsg = "[翻译错误]:翻译文件错误"
					return
				}
				if file == "string.xlsx" || file == "error.xlsx" {
					re = regexp.MustCompile(`%[a-z]`) // 匹配占位符
				}
				break
			}
		}
	}

	if !checkOnly {
		sliceLen := len(workSheet) * (len(Field) + 2)
		rowsSlice = make([]string, 0, sliceLen)
		rowsSlice = append(rowsSlice, "--Don't Edit!!!") //第一行先占用

		// 配置
		rowsSlice = append(rowsSlice, "return {")
		rowsSlice = append(rowsSlice, "{")
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
					if idx == 0 { // key 字段
						id = text
						if len(id) == 0 { //如果key字段的值为空，则停止解析配置
							break forLable
						} else {
							if _, ok := idMap[id]; ok {
								errInfo.Level = E_ERROR
								errInfo.ErrMsg = fmt.Sprintf("[配置错误 id=%s,行=%s]:id重复", id, i+1)
								return
							} else {
								idMap[text] = true
							}
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

					if len(text) <= 0 {
						continue
					}

					// json格式是否正确
					if f.Type == "table" {
						err := checkJson(text)
						if err != nil {
							errInfo.Level = E_ERROR
							errInfo.ErrMsg = fmt.Sprintf("[配置错误 id=%s,字段=%s]:json格式错误", id, f.Name)
							return
						}
					}

					// 只生成服务器需要的字段
					if !checkOnly && (f.Mode == "s" || f.Mode == "d") {
						baseText := text // 替换成翻译内容
						if needTrans && (f.Type == "table" || f.Type == "object" || f.Type == "string") {
							trCell := getLangCell(langSheet, id, f.Name)
							if len(trCell) > 0 {
								// 对比英文字符对不对
								if len(specFile) > 0 && !checkAscii(text, trCell) {
									errInfo.Level = E_ERROR
									errInfo.ErrMsg = fmt.Sprintf("[翻译错误 id=%s,字段=%s]:翻译内容不匹配,源=%s,翻译=%s", id, f.Name, text, trCell)
									return
								}
								text = trCell
							} else {
								if IsChineseChar(text) {
									errInfo.Level = E_ERROR
									errInfo.ErrMsg = fmt.Sprintf("[翻译错误 id=%s,字段=%s]:翻译缺失", id, f.Name)
								}
							}
						}

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
								if needTrans && re != nil {
									reSlice1 := re.FindAllString(baseText, -1)
									reSlice2 := re.FindAllString(text, -1)
									result, errstr := checkTranslation(reSlice1, reSlice2)
									if !result {
										errInfo.Level = E_ERROR
										errInfo.ErrMsg = fmt.Sprintf("[翻译错误 id=%s,字段=%s]:翻译占位符错误,err=%s", id, f.Name, errstr)
										return
									}
								}
								text = strings.Replace(text, "'", `\'`, -1) // 替换单引号
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
	sheetName := xlFile.GetSheetName(1)
	workSheet := xlFile.GetRows(sheetName) //sheetName = "Sheet1"
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

			if len(specFile) == 0 || (len(specFile) > 0 && f.Name() == specFile) {
				xlsxSlice = append(xlsxSlice, XlsxPath{path, f.Name(), uint64(f.ModTime().UnixNano() / 1000000)})
			}
			return nil
		}
		return mErr
	})
	checkErr(err)
}

func findLangDir() error {
	if len(langRoot) > 0 {
		dir, err := ioutil.ReadDir(langRoot)
		if err != nil {
			return err
		}

		var realDir string
		for _, fi := range dir {
			if fi.IsDir() {
				dirRoot := langRoot + "/" + fi.Name()
				filepath.Walk(dirRoot, func(path string, f os.FileInfo, err error) error {
					if !f.IsDir() {
						ok, mErr := filepath.Match("$*.xlsx", f.Name())
						if ok {
							realDir = fi.Name()
							return nil
						}
						return mErr
					}
					return nil
				})
			}
		}

		if len(realDir) > 0 {
			langRoot += ("/" + realDir)
			color.Yellow("翻译文件目录： %s", langRoot)
			return nil
		}
		return errors.New("翻译目录错误")
	}
	return nil
}

func loadLastModTime() {
	_, err := os.Stat(luaRoot)
	notExist := os.IsNotExist(err)
	if notExist {
		// 输出文件夹不存在
		needForce = true
		return
	}

	file, ferr := os.Open("lastModTime.txt")
	if ferr != nil {
		needForce = true
		if runtime.GOOS == "windows" {
			os.RemoveAll(luaRoot)
		} else {
			if !notExist {
				cmdstr := fmt.Sprintf(`find %s -type f -name "*.lua"|xargs rm -rf`, luaRoot)
				cmd := exec.Command("sh", "-c", cmdstr)
				err = cmd.Run()
				checkErr(err)
			}
		}
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

type LangSheet struct {
	SheetRows [][]string     // 翻译文件xlsx数据
	FieldRef  map[string]int // 翻译字段反索引
	IdRef     map[string]int // 翻译id反索引
}

var luaRoot string
var xlsxRoot string
var langRoot string // 翻译文件根目录
var specFile string // 生成指定的文件
var resChan chan Result
var xlsxSlice []XlsxPath
var lastModTime map[string]uint64
var needForce bool = false //是否需要全部重新生成

func main() {
	flag.StringVar(&xlsxRoot, "i", "", "输入路径")
	flag.StringVar(&luaRoot, "o", "", "输出路径")
	flag.StringVar(&langRoot, "l", "", "翻译文件路径")
	flag.StringVar(&specFile, "file", "", "指定生成某个文件")
	flag.Parse()
	if len(luaRoot) == 0 || len(xlsxRoot) == 0 {
		color.Red("输入路径或输出路径为空")
		return
	}

	err := findLangDir()
	if err != nil {
		color.Red(err.Error())
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
	var changedXlsx []XlsxPath
	if needForce {
		color.Red("注意：全部重新生成!!!")
		changedXlsx = xlsxSlice
	} else {
		changedXlsx = make([]XlsxPath, 0, len(xlsxSlice))
		for _, xp := range xlsxSlice {
			lastModTm, ok := lastModTime[xp.PathName]
			if ok && lastModTm == xp.ModTime {
				//xlsxSlice = append(xlsxSlice[:i], xlsxSlice[i+1:]...)
				//fmt.Println(xp.PathName + "无变化")
			} else {
				color.Red("[有变化的文件]： " + xp.PathName)
				changedXlsx = append(changedXlsx, xp)
			}
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
		saveLastModTime()
	} else {
		color.Set(color.FgRed)
		fmt.Printf("生成有错误，条目=%d，耗时=%d 毫秒 ！\n", len(resSlice), msec)
		color.Unset()
	}
}
