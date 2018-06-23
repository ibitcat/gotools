// 生成 typescript
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/vmihailenco/msgpack"
)

var cliDats []CliDat

// 解析成lua格式
func (c *XlsxConv) parseToTsDat(workSheet [][]string) {
	// 检查key字段
	keyField := c.Fields[0]
	if keyField.Mode != "c" && keyField.Mode != "d" {
		return
	}

	var cliDat CliDat
	var writeBytes bytes.Buffer
	msgEnc := msgpack.NewEncoder(&writeBytes)

	// 生成interface字段，load行
	tname := strings.TrimSuffix(c.FileName, ".xlsx")
	cliDat.Name = tname
	interfaceName := string(strings.ToUpper(string(([]byte(tname))[0]))+string(([]byte(tname))[1:])) + "Cfg"
	cliDat.Interface = append(cliDat.Interface, "\texport interface "+interfaceName+" {")
	exportLine := "\texport let " + tname + ": {[id: number]: " + interfaceName + "} = {}"
	loadLine := "\tParser.add(" + tname + ", '" + tname + "', ["

	var keys []int
	for k := range c.Fields {
		field := c.Fields[k]
		if field.Mode != "c" && field.Mode != "d" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for i, k := range keys {
		field := c.Fields[k]
		line := "\t\treadonly " + field.Name + ": "
		switch field.Type {
		case "table", "array":
			line += "Array<any>"
		case "object":
			line += "any"
		case "int", "number":
			line += "number"
		default:
			line += field.Type
		}
		cliDat.Interface = append(cliDat.Interface, line)
		loadLine += "['" + field.Name + "','" + field.Type + "']"
		if i < len(keys)-1 {
			loadLine += ","
		}
	}
	cliDat.Interface = append(cliDat.Interface, "\t}")
	cliDat.Export = append(cliDat.Export, exportLine)
	cliDat.Load = append(cliDat.Load, loadLine+"])")

	// 从配置的第4行开始解析
	msgEnc.Encode(tname)
	validLine := make([]int, 0, len(workSheet)-4)
forLable:
	for i := 4; i < len(workSheet); i++ {
		row := workSheet[i]
		if len(row) <= 0 {
			continue
		}
		if len(row[0]) == 0 {
			break forLable
		}
		validLine = append(validLine, i)
	}
	msgEnc.Encode(len(validLine))

	for _, i := range validLine {
		row := workSheet[i]
		for idx, text := range row {
			f, ok := c.Fields[idx]
			if !ok {
				continue
			}
			if f.Mode != "c" && f.Mode != "d" {
				continue
			}

			var v interface{}
			switch f.Type {
			case "table", "array", "object":
				if text == "" {
					v = ""
				} else {
					json.Unmarshal([]byte(text), &v)
				}
			case "int", "number":
				v, _ = strconv.ParseFloat(text, 64)
			case "string":
				v = text
			case "boolean", "bool":
				if text == "" {
					v = false
				} else {
					v, _ = strconv.ParseBool(text)
				}
			default:
				v = text
				c.Errs = append(c.Errs, ErrorInfo{E_WARN, "[Ts]未知数据类型: " + f.Type})
			}

			msgEnc.Encode(v)
		}
	}

	cliDat.msgBytes = writeBytes.Bytes()
	cliDats = append(cliDats, cliDat)
}

func zlibCompress(src []byte) []byte {
	var in bytes.Buffer
	w := zlib.NewWriter(&in)
	w.Write(src)
	w.Close()
	return in.Bytes()
}

func genCliTsFile() {
	_, err := os.Stat(tsAbsRoot)
	if os.IsNotExist(err) {
		err = os.MkdirAll(tsAbsRoot, os.ModePerm)
		if err != nil {
			return
		}
	}

	sort.Slice(cliDats, func(i, j int) bool { return strings.Compare(cliDats[i].Name, cliDats[j].Name) < 0 })

	luaFile := fmt.Sprintf("%s/Config.ts", tsAbsRoot)
	outFile, operr := os.OpenFile(luaFile, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
	if operr != nil {
		panic(operr)
	}
	defer outFile.Close()

	outFile.WriteString("module Config {\n")
	for i := 0; i < len(cliDats); i++ {
		outFile.WriteString(strings.Join(cliDats[i].Interface, "\n"))
		outFile.WriteString("\n")
	}
	outFile.WriteString("\n")
	for i := 0; i < len(cliDats); i++ {
		outFile.WriteString(strings.Join(cliDats[i].Export, "\n"))
		outFile.WriteString("\n")
	}
	outFile.WriteString("\n")
	for i := 0; i < len(cliDats); i++ {
		outFile.WriteString(strings.Join(cliDats[i].Load, "\n"))
		outFile.WriteString("\n")
	}
	outFile.WriteString("}\n")
	outFile.Sync()

	datPath := fmt.Sprintf("%s/data.dat", tsAbsRoot)
	datFile, err := os.OpenFile(datPath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	defer datFile.Close()
	var writeBytes [][]byte
	for i := 0; i < len(cliDats); i++ {
		writeBytes = append(writeBytes, cliDats[i].msgBytes)
	}
	datFile.Write(zlibCompress(bytes.Join(writeBytes, []byte{})))
	datFile.Sync()
}
