package main

// error level
const (
	E_NONE   = iota
	E_NOTICE //通知
	E_WARN   //警告
	E_ERROR  //错误
)

type ErrorInfo struct {
	Level  int
	ErrMsg string
}

type FieldInfo struct {
	Name string // 字段名
	Type string // 字段类型
	Mode string // 生成方式(s=server,c=client,d=双端,r=策划)
}

// 一个xlsx转换
type XlsxConv struct {
	AbsPath     string            // 文件绝对路径
	RelPath     string            // 文件相对路径(例如task\task.xlsx)
	FolderName  string            // 文件夹路径(例如task\)
	FileName    string            // 文件名（例如：tast.xlsx）
	ModTime     uint64            // 文件修改时间（毫秒）
	Fields      map[int]FieldInfo // 字段信息
	hasSrvField bool              // 是否有服务器字段
	Msec        int               // 耗时（毫秒）
	Errs        []ErrorInfo       // 错误信息
	checkOnly   bool              // 仅仅检查配置错误，并不生成
}

type CliDat struct {
	Name      string
	Interface []string
	Export    []string
	Load      []string
	msgBytes  []byte
}
