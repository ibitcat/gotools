--$Id$

-- 树型打印一个 table,不用担心循环引用
table.print = function(root,depthMax,excludeKey,excludeType)
	assert(type(root)=="table","无法打印非table")
	depthMax = depthMax or 3 -- 默认三层
	local cache = { [root] = "." }
	local depth = 0
	print("{")
	local function _dump(t,space,name)
		local temp = {}
		for k,v in pairs(t) do
			local key = tostring(k)
			if type(k) == "string" then
				key ='\"' .. tostring(k) .. '\"'
			end

			if type(v) == "table" then
				if cache[v] then
					table.insert(temp,space .. "["..key.."]" .." = " .. " {" .. cache[v].."},")
				else
					local new_key = name .. "." .. tostring(k)
					cache[v] = new_key .." ->[".. tostring(v) .."]"

					-- table 深度判断
					depth = depth + 1
					if depth>=depthMax or (excludeKey and excludeKey==k) then
						table.insert(temp,space .. "["..key.."]" .." = " .. "{ ... }")
					else
						local tableStr = _dump(v,space .. (next(t,k) and "|" or " ") .. string.rep(" ",#key<4 and 4 or #key),new_key)
						if tableStr then		-- 非空table
							table.insert(temp,space .. "["..key.."]" .." = " .. "{")
							table.insert(temp, tableStr)
							table.insert(temp,space .."},")
						else 						-- 空table
							table.insert(temp,space .. "["..key.."]" .." = " .. "{ },")
						end
						--table.insert(temp, _dump(v,space .. (next(t,k) and "|" or " " ).. string.rep(" ",#key),new_key))
					end
					depth = depth -1
				end
			else
				local vType = type(v)
				if not excludeType or excludeType~=vType then
					if vType == "string" then
						v = '\"' .. v .. '\"'
					else
						v = tostring(v) or "nil"
					end
					table.insert(temp,space .. "["..key.."]" .. " = " .. v ..",")
					--tinsert(temp,"+" .. key .. " [" .. tostring(v).."]")
				end
			end
		end

		return #temp>0 and table.concat(temp,"\n") or nil
	end
	local allTableString = _dump(root, "    ","")
	print(allTableString or "")
	print("}")
	return allTableString
end

function xprint( ... )
	for i=1,arg.n do
		local value = arg[i]
		if type(value) == "table" then
			table.print(value)
		elseif type(value) == "string" then
			print('\"' .. value .. '\"')
		else
			print(tostring(value))
		end
	end
	print("\n")
end


require("msgpack")
print('--------------------------pack 和 unpack')
-- pack 和 unpack
local pkt  = msgpack.newPacket()
pkt:packs(1,2,3,"hello",{"a","b","c"})
xprint(pkt:unpacks()) --解包所有的元素

pkt:reread()	--重读pkt
local item1,item2 = pkt:unpacks(2)	--解包前2两个元素，输出：1,2
-- unpacks() 不传参数表示解包所有元素，传一个数字参数，表示解包指定个数的元素


print('--------------------------wrap 和 unwrap')
-- wrap 和 unwrap
local pkt = msgpack.newPacket()
pkt:wrap(1,2,3)
local item1,item2,item3 = pkt:unwrap()
print(item1,item2,item3)


print('--------------------------packPacket 和 unpackPacket')
-- packPacket 和 unpackPacket
local pkt1 = msgpack.newPacket()
pkt1:packs(1,2,3,"haha")
local pkt = msgpack.newPacket()
pkt:packs("hello",1000) --可以pack基础类型的元素
pkt:packPacket(pkt1)

local item1,item2 = pkt:unpacks(2) --取出前两个基础类型元素
local subPkt = msgpack.newPacket()
pkt:unpackPacket(subPkt)	--取出子包
print(item1,item2, subPkt:unpacks())		--解子包


print('--------------------------wrap_pre、wrap_end 和 unwrap_header')
-- wrap_pre、wrap_end 和 unwrap_header
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