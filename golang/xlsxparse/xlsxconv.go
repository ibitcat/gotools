// 一次xlsx转换

package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/fatih/color"
)

var ConvChan chan *XlsxConv

func (c *XlsxConv) hasError(lv int) bool {
	if c.Errs == nil {
		return false
	}
	for _, e := range c.Errs {
		if e.Level >= lv {
			return true
		}
	}
	return false
}

func (c *XlsxConv) formatErr() string {
	var str string
	if c.Errs != nil {
		warnStr := make([]string, 0, len(c.Errs))
		errStr := make([]string, 0, len(c.Errs))
		for _, e := range c.Errs {
			if e.Level == E_WARN {
				warnStr = append(warnStr, e.ErrMsg)
			} else if e.Level == E_ERROR {
				errStr = append(errStr, e.ErrMsg)
			}
		}

		if len(warnStr) > 0 {
			str += fmt.Sprintf("[警告%d条]：\r\n", len(warnStr)) + strings.Join(warnStr, "\r\n")
		}
		if len(errStr) > 0 {
			str += fmt.Sprintf("[错误%d条]：\r\n", len(errStr)) + strings.Join(errStr, "\r\n")
		}
	}
	return str
}

func (c *XlsxConv) loadXlsxHead(workSheet [][]string) {
	if len(workSheet) < 4 {
		c.Errs = append(c.Errs, ErrorInfo{E_ERROR, "配置头至少要4行"})
		return
	}

	fieldRow := workSheet[1] //字段名
	typeRow := workSheet[2]  //字段类型
	modeRow := workSheet[3]  //生成方式
	fieldNum := len(fieldRow)
	if fieldNum == 0 {
		c.Errs = append(c.Errs, ErrorInfo{E_ERROR, "[配置头错误]:配置字段名为空"})
		return
	}

	fieldCheck := make(map[string]bool, fieldNum) //检查字段是否重复
	c.Fields = make(map[int]FieldInfo, fieldNum)
	for i, fieldName := range fieldRow {
		fieldType := typeRow[i]
		modeType := modeRow[i]
		if modeType == "s" || modeType == "d" {
			c.hasSrvField = true
		}

		// 检测key字段
		if i == 0 { //字段行的第一个字段为配置的key,需要检查下
			if len(fieldName) == 0 {
				c.Errs = append(c.Errs, ErrorInfo{E_ERROR, "[配置头错误]:配置没有key"})
			}
			if fieldType != "int" && fieldType != "string" {
				c.Errs = append(c.Errs, ErrorInfo{E_ERROR, "[配置头错误]:id字段类型错误"})
			}
		}

		// 字段通用检查
		if modeType == "r" {
			continue
		} else {
			var errStr string
			if len(fieldName) > 0 {
				if strings.Contains(fieldName, " ") {
					errStr = fmt.Sprintf("[配置头错误]:字段[%s]有空格", fieldName)
					c.Errs = append(c.Errs, ErrorInfo{E_ERROR, errStr})
				}
				if modeType != "c" && modeType != "s" && modeType != "d" {
					errStr = fmt.Sprintf("[配置头错误]:字段[%s]生成方式错误", fieldName)
					c.Errs = append(c.Errs, ErrorInfo{E_ERROR, errStr})
				}
				if len(fieldType) == 0 {
					errStr = fmt.Sprintf("[配置头错误]:字段[%s]类型不存在", fieldName)
					c.Errs = append(c.Errs, ErrorInfo{E_ERROR, errStr})
				}
				if _, ok := fieldCheck[fieldName+modeType]; ok {
					errStr = fmt.Sprintf("[配置头错误]:字段[%s]重复", fieldName)
					c.Errs = append(c.Errs, ErrorInfo{E_ERROR, errStr})
				}
			} else {
				if len(modeType) > 0 || len(fieldType) > 0 {
					errStr = fmt.Sprintf("[配置头错误]:第[%d]个字段名为空", i+1)
					c.Errs = append(c.Errs, ErrorInfo{E_ERROR, errStr})
				}
			}
			if len(errStr) == 0 {
				c.Fields[i] = FieldInfo{fieldName, fieldType, modeType}
				fieldCheck[fieldName+modeType] = true
			}
		}
	}
}

func (c *XlsxConv) generate() {
	startTime := time.Now()
	c.Errs = make([]ErrorInfo, 0, 5)
	defer func() {
		if err := recover(); err != nil {
			c.Errs = append(c.Errs, ErrorInfo{E_ERROR, fmt.Sprintf("%v", err)})
		}
		c.Msec = int(time.Now().Sub(startTime).Nanoseconds() / 1e6)
		ConvChan <- c
	}()

	xlFile, err := excelize.OpenFile(c.AbsPath)
	if err != nil {
		c.Errs = append(c.Errs, ErrorInfo{E_ERROR, err.Error()})
		return
	}

	// 第一个sheet为配置
	sheetName := xlFile.GetSheetName(1)
	workSheet := xlFile.GetRows(sheetName) //sheetName = "Sheet1"
	c.loadXlsxHead(workSheet)
	if c.hasError(E_WARN) {
		return
	}

	if len(tsAbsRoot) > 0 {
		// typescript
		c.parseToTsDat(workSheet)
	} else {
		// lua
		c.parseToLua(workSheet)
	}
}

// 开始生成
func startConv() {
	loadLastModTime()
	err := WalkXlsx(xlsxRoot)
	if err != nil {
		log.Println(err.Error())
		return
	}

	count := len(ConvTasks)
	if count > 0 {
		if len(LastModifyTime) == 0 {
			color.Red("注意：全部重新生成!!!")
		}
		fmt.Println("正在生成，请稍后……,条目=", count)

		startTime := time.Now()
		ConvChan = make(chan *XlsxConv, count)
		for _, conv := range ConvTasks {
			go conv.generate()
		}

		for i := 0; i < count; i++ {
			<-ConvChan
		}

		if len(tsAbsRoot) > 0 {
			genCliTsFile()
		} else {
			saveConvTime()
		}

		printResult(startTime)
	} else {
		color.Green("无需生成^_^")
	}
}

func saveConvTime() {
	file := curAbsRoot + "\\" + "lastModTime.txt"
	outFile, operr := os.OpenFile(file, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
	if operr != nil {
		fmt.Println("创建[lastModTime.txt]文件出错", "错误")
	}
	defer outFile.Close()

	for _, conv := range ConvTasks {
		if !conv.hasError(E_ERROR) {
			LastModifyTime[conv.RelPath] = conv.ModTime
		}
	}

	// save time
	modTimes := make([]string, 0, len(LastModifyTime))
	for name, tm := range LastModifyTime {
		modTimes = append(modTimes, name+"|"+strconv.FormatUint(tm, 10))
	}
	outFile.WriteString(strings.Join(modTimes, "\n"))
	outFile.Sync()
}

func printResult(startTime time.Time) {
	msec := GetDurationMs(startTime)
	color.Green("生成结束，结果如下：")

	result := 0 //0=全部ok，1=有警告，2=有错误
	for _, conv := range ConvTasks {
		//fmt.Println(strings.Repeat("-", 65))
		fmt.Printf("%-60s |%5sms\n", conv.RelPath, strconv.Itoa(conv.Msec))
		for _, errinfo := range conv.Errs {
			switch errinfo.Level {
			case E_NOTICE:
				{ //通知
					color.Set(color.FgCyan)
					fmt.Printf(">%s\n", errinfo.ErrMsg)
					color.Unset()
				}
			case E_WARN:
				{ //警告
					color.Set(color.FgYellow)
					fmt.Printf(">%s\n", errinfo.ErrMsg)
					color.Unset()
					result |= 1
				}
			case E_ERROR:
				{ //错误
					result |= 2
					color.Set(color.FgRed)
					fmt.Printf(">%s\n", errinfo.ErrMsg)
					color.Unset()
				}
			}
		}
	}

	if result == 0 {
		color.Set(color.FgGreen)
		fmt.Printf("生成完毕，条目=%d，耗时=%d 毫秒 ！\n", len(ConvTasks), msec)
	} else if result == 1 {
		color.Set(color.FgYellow)
		fmt.Printf("生成有警告，条目=%d，耗时=%d 毫秒 ！\n", len(ConvTasks), msec)
	} else {
		color.Set(color.FgRed)
		fmt.Printf("生成有错误，条目=%d，耗时=%d 毫秒 ！\n", len(ConvTasks), msec)
	}
	color.Unset()
}
