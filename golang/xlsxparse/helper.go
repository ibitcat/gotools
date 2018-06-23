package main

import (
	"encoding/json"
	"errors"
	"time"
	"unicode"
)

// 获取两个时间戳的毫秒差
func GetDurationMs(t time.Time) int {
	return int(time.Now().Sub(t).Nanoseconds() / 1e6)
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

// 实验性特性
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
			return false
		}
	}
	return true
}

func isChineseChar(str string) bool {
	for _, r := range str {
		if unicode.Is(unicode.Scripts["Han"], r) {
			return true
		}
	}
	return false
}
