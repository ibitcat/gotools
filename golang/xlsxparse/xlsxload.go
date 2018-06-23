// 加载xlsx配置文件
package main

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// xlsx 文件最后修改时间戳（毫秒）
var LastModifyTime map[string]uint64
var ConvTasks []*XlsxConv

func loadLastModTime() {
	LastModifyTime = make(map[string]uint64, 500)
	file, err := os.Open(curAbsRoot + "\\lastModTime.txt")
	if err != nil {
		if runtime.GOOS == "windows" {
			os.RemoveAll(luaAbsRoot)
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
				LastModifyTime[s[0]] = tm
			}
		}
	}
}

// load xlsx
func WalkXlsx(dir string) error {
	_, err := os.Stat(dir)
	notExist := os.IsNotExist(err)
	if notExist {
		return err
	}

	_getLastConvTime := func(name string) uint64 {
		tm, ok := LastModifyTime[name]
		if ok {
			return tm
		}
		return 0
	}

	err = filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}

		ok, mErr := filepath.Match("[^~$]*.xlsx", f.Name())
		if ok {
			modifyTime := uint64(f.ModTime().UnixNano() / 1000000)
			relpath := strings.TrimPrefix(path, dir+"\\")
			if _getLastConvTime(relpath) != modifyTime {
				conv := &XlsxConv{
					AbsPath:  path,
					RelPath:  relpath,
					FileName: f.Name(),
					ModTime:  modifyTime,
				}
				conv.FolderName = strings.TrimSuffix(relpath, conv.FileName)
				ConvTasks = append(ConvTasks, conv)
			}
			return nil
		}
		return mErr
	})
	if err == nil {
		sort.Slice(ConvTasks, func(i, j int) bool { return ConvTasks[i].RelPath < ConvTasks[j].RelPath })
	}
	return err
}
