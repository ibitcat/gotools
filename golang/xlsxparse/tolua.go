// 生成lua格式

package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

func parseLuaFeild(idx int, text string, f *FieldInfo) string {
	isSvrField := (f.Mode == "s" || f.Mode == "d") // 服务器字段
	var str string
	if isSvrField {
		if f.Type == "int" || f.Type == "number" {
			str = fmt.Sprintf("        ['%s'] = %s,", f.Name, text)
		} else {
			if f.Type == "table" {
				// 压缩成一行
				text = strings.Replace(text, " ", "", -1)
				text = strings.Replace(text, "\n", "", -1)
			} else if f.Type == "string" { // ' 替换成 \'
				text = strings.Replace(text, "'", `\'`, -1) // 替换单引号
			}
			str = fmt.Sprintf("        ['%s'] = '%s',", f.Name, text)
		}

		if idx == 0 {
			if f.Type == "int" {
				str = fmt.Sprintf("    [%s] = {\n", text) + str
			} else {
				str = fmt.Sprintf("    ['%s'] = {\n", text) + str
			}
		}
	}
	return str
}

func (c *XlsxConv) parseLuaFooter(rowsSlice []string) []string {
	rowsSlice = append(rowsSlice, "},")
	// 字段table
	idxs := make([]int, 0, len(c.Fields))
	for k, _ := range c.Fields {
		idxs = append(idxs, k)
	}
	sort.Ints(idxs)
	rowsSlice = append(rowsSlice, "\n--field")
	rowsSlice = append(rowsSlice, "{")
	for _, v := range idxs {
		f := c.Fields[v]
		if f.Mode == "s" || f.Mode == "d" {
			rowsSlice = append(rowsSlice, fmt.Sprintf("    ['%s'] = '%s',", f.Name, f.Type))
		}
	}
	rowsSlice = append(rowsSlice, "},\n")

	// key
	rowsSlice = append(rowsSlice, fmt.Sprintf("'%s'", c.Fields[0].Name))
	rowsSlice = append(rowsSlice, "}")
	return rowsSlice
}

func (c *XlsxConv) outPutToLuaFile(rowsSlice []string) {
	outDir := luaAbsRoot + "\\" + c.FolderName
	_, err := os.Stat(outDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(outDir, os.ModePerm)
		if err != nil {
			c.Errs = append(c.Errs, ErrorInfo{E_ERROR, err.Error()})
			return
		}
	}

	name := strings.TrimSuffix(c.FileName, ".xlsx")
	file := fmt.Sprintf("%s%s.lua", outDir, name)
	outFile, operr := os.OpenFile(file, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
	if operr != nil {
		c.Errs = append(c.Errs, ErrorInfo{E_ERROR, operr.Error()})
		return
	}
	defer outFile.Close()

	outFile.WriteString(strings.Join(rowsSlice, "\n"))
	outFile.Sync()
}

func (c *XlsxConv) parseToLua(workSheet [][]string) {
	c.checkOnly = false

	// 检查key字段
	keyField := c.Fields[0]
	if keyField.Mode != "s" && keyField.Mode != "d" {
		c.checkOnly = true
		c.Errs = append(c.Errs, ErrorInfo{E_NOTICE, "不需要生成"})
	}

	var rowsSlice []string = nil
	if !c.checkOnly {
		sliceLen := len(workSheet) * (len(c.Fields) + 10)
		rowsSlice = make([]string, 0, sliceLen)
		rowsSlice = append(rowsSlice, "--Don't Edit!!!")
		rowsSlice = append(rowsSlice, "return {")
		rowsSlice = append(rowsSlice, "{")
	}

	// 从配置的第4行开始解析
	idMap := make(map[string]bool) //检查id是否重复
	_parseRow := func(i int, row []string) int {
		var id string
		var errStr string
		rowLen := 0
		for idx, text := range row {
			rowLen += len(text)
			if f, ok := c.Fields[idx]; ok {
				if idx == 0 {
					id = text
					if len(id) > 0 {
						if _, ok := idMap[id]; ok {
							errStr = fmt.Sprintf("[id重复 id=%s,行=%s]", id, i+1)
							c.Errs = append(c.Errs, ErrorInfo{E_ERROR, errStr})
							c.checkOnly = true
						} else {
							idMap[id] = true
						}
					}
				}
				if len(text) <= 0 {
					continue
				}

				// 替换翻译
				/*if c.Lang != nil {
					if langText := c.getLangCellText(id, f); len(langText) > 0 {
						if c.checkLangText(langText, text, id, f.Name) {
							text = langText
						}
					}
				}*/

				// json格式是否正确
				if f.Type == "table" || f.Type == "object" {
					err := checkJson(text)
					if err != nil {
						errStr = fmt.Sprintf("[json格式错误 id=%s,字段=%s]", id, f.Name)
						c.Errs = append(c.Errs, ErrorInfo{E_ERROR, errStr})
						c.checkOnly = true
					}
				}

				if !c.checkOnly {
					if str := parseLuaFeild(idx, text, &f); len(str) > 0 {
						rowsSlice = append(rowsSlice, str)
					}
				}
			}
		}

		if rowLen > 0 {
			if len(id) == 0 {
				errStr = fmt.Sprintf("[id为空,行=%d]", i+1)
				c.Errs = append(c.Errs, ErrorInfo{E_ERROR, errStr})
				c.checkOnly = true
			}
			if !c.checkOnly {
				rowsSlice = append(rowsSlice, "    },")
			}
		}
		return rowLen
	}

	for i := 4; i < len(workSheet); i++ {
		row := workSheet[i]
		if len(row) > 0 {
			if rowLen := _parseRow(i, row); rowLen <= 0 {
				break
			}
		}
	}

	// 配置解析完毕，添加字段表
	if !c.checkOnly {
		rowsSlice = c.parseLuaFooter(rowsSlice)
		c.outPutToLuaFile(rowsSlice) // 生成lua文件
	}
}
