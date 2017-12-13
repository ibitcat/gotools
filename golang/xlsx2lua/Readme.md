## excel 转lua配置

xlsx 文件格式如下图：  
![模板](xlsx.jpg)  

**特性：**

- xlsx生成lua文件
- 支持id重复检测
- 支持json格式检测  
~~- 支持md5值检测文件是否有变动（因为会拖累生成速度，干脆暴力，每次全都生成一次）~~

**配置头部定义：**  

- 第一行为字段注释
- 第二行为字段名，英文开头，不允许带空格，不允许出现中文和特殊符号
- 第三行为字段类型，包括 int，number，table，string，注意：table为json字符串，例如，[["hit",8,10],["cri",8,10]] ，表示一个json数组
- 第四行为生成模式，c(client)表示客户端专用，s(server)表示服务器专用，d(double)表示双端都要用，r(remark)表示双端都不需要生成


**参数：**

- -i xxx 表示xlsx文件路径
- -o yyy 表示生成lua文件的路径
- -l aaa 表示翻译文件路径 （为了本地化，不填则不翻译）
- -file bbb.xlsx 表示只生成指定excel文件

> 翻译的原理是：在生成lua的同时，检查字段是否是string并且模式是s或者d，然后检查该字段是否有相应的翻译字符串对应，如果有则替换。

例子：

***windows：***  
`xlsx2lua.exe -i xlsx -o l-xlsx -l language`

表示 将`xlsx/`文件夹下的所有xlsx装换成lua，lua文件的输出路径为`/l-xlsx`,并且在转换的同时翻译相关字段，翻译文件的路径为：`language`


***linux:***  
`xlsx2lua.exe -i xlsx -o l-xlsx -l language`

注意：linux与windows区别在，路径斜杠和反斜杠的问题，请注意。


生成的lua文件格式如下：

实例： `material.lua`

```lua
return {
-- 具体的配置内容
{
    [202000] = {
        ['id'] = 202000,
        ['name'] = '各部位装备锻造书',
        ['mainclass'] = 2,
        ['subclass'] = 2,
        ['turnLevel'] = 0,
        ['level'] = 1,
        ['quality'] = 4,
        ['abandon'] = 1,
        ['overlay'] = 99,
        ['sellMoney'] = 100,
    },
    [201057] = {
        ['id'] = 201057,
        ['name'] = '20级主手锻造书',
        ['mainclass'] = 2,
        ['subclass'] = 2,
        ['turnLevel'] = 0,
        ['level'] = 20,
        ['quality'] = 3,
        ['abandon'] = 1,
        ['overlay'] = 99,
    },
},

-- 服务器用到的字段
{
    ['id'] = 'int',
    ['name'] = 'string',
    ['mainclass'] = 'int',
    ['subclass'] = 'int',
    ['turnLevel'] = 'int',
    ['level'] = 'int',
    ['quality'] = 'int',
    ['abandon'] = 'int',
    ['timeLimit'] = 'int',
    ['overlay'] = 'int',
    ['sortweight'] = 'int',
    ['alchemy'] = 'int',
    ['exp'] = 'int',
    ['glyphExp'] = 'int',
    ['sellMoney'] = 'int',
},

-- 配置主键
'id'
}
```

## 压缩
go生成的可执行文件有点大，可以采用下面两个步骤减小二进制文件大小。

1. 在build时，加入参数`go build -ldflags "-w -s"` 
2. 使用upx加壳压缩

## log

- 2017-10-23

	1. 增加新的字段生成方式r（remark）,表示这个字段服务器和客户端都不需要生成，只是用来做备注，给策划或开发看的。

- 2017-11-27

	1. 优化翻译处理，工具自动搜索翻译文件目录

- 2017-11-30

	1. 检查翻译是否缺失
	2. 修复不翻译table类型字段的bug

- 2017-12-11

	1. 翻译错误（占位符错误）更加详细
	2. 检查翻译文件的英文字符是否与源文件中的英文字符相匹配

- 2017-12-12

	1. 支持 `-file test.xlsx`参数，生成指定xlsx文件的配置