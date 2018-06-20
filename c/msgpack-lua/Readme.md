
基于[lua-cmsgpack](https://github.com/antirez/lua-cmsgpack)修改，增加了一些增强特性。

## 安装

可以集成在项目内，也可以生成动态库。

~~~
sh build.sh
~~~

## 使用

### pack 和 unpack

~~~lua
require("msgpack")
local pkt  = msgpack.newPacket()
pkt:packs(1,2,3,"hello",{"a","b","c"})
print(pkt:unpacks()) --解包所有的元素

pkt:reread()	--重读pkt
local item1,item2 = pkt:unpacks(2)	--解包前2两个元素，输出：1,2
-- unpacks() 不传参数表示解包所有元素，传一个数字参数，表示解包指定个数的元素
~~~

### wrap 和 unwrap

wrap(1,2,3) 等价于 pkt:packs({1,2,3})，即是一种打包数组的变种，wrap的方式不需要构建一个table，可以减少一些table。

~~~lua
local pkt = msgpack.newPacket()
pkt:wrap(1,2,3)
local item1,item2,item3 = pkt:unwrap()
print(item1,item2,item3)
~~~

### packPacket 和 unpackPacket

可以用来pack一个子包（也是一个msgpack对象），底层是用bytes来存储数据。

~~~lua
local pkt1 = msgpack.newPacket()
pkt1:packs(1,2,3,"haha")
local pkt = msgpack.newPacket()
pkt:packs("hello",1000) --可以pack基础类型的元素
pkt:packPacket(pkt1)

local item1,item2 = pkt:unpacks(2) --取出前两个基础类型元素
local subPkt = msgpack.newPacket()
pkt:unpackPacket(subPkt)	--取出子包
print(subPkt:unpacks())		--解子包
~~~

### wrap_pre、wrap_end 和 unwrap_header

用来增强wrap，一个wrap_pre表示pkt要开始打包一个数组格式的元素，wrap_end 表示结束数组打包，unwrap_header则表示解包数组的长度。

~~~lua
-- 假如需要打包下面的数据：
-- t = {1,2,3,{{"a","b"},{100,200}},"hello","domi"}

local pkt = msgpack.newPacket()
pkt:wrap_pre()
	pkt:packs(1,2,3)
	pkt:wrap_pre()
		pkt:wrap("a","b")
		pkt:wrap(100,200)
	pkt:wrap_end()
	pkt:packs("hello")
	pkt:packs("domi")
pkt:wrap_end()

local n = pkt:unwrap_header()
print(n) --n=6
local item1,item2,items = pkt:unpacks(3) --输出：1,2,3
print(item1,item2,items)
local sn = pkt:unwrap_header()
for j=1,sn do
	print(pkt:unwrap())
end
local hello,domi = pkt:unpacks(2)
print(hello,domi)
~~~
