-- lua 打印table结构，参考云风的思路，稍微优化了一下
-- 加了层级判断，因为工作中有些table太多层嵌套了，导致关键信息看不到

local _print = _G.print

-- local var
local _eqStr = " = "
local _bktL = "{"
local _bktR = "}"
local _tbShort = "{ ... }"
local _tbEmpty = "{ }"
local _indent = "    "

local function _EscapeKey(key)
	if type(key) == "string" then
		key ='\"' .. tostring(key) .. '\"'
	else
		key = tostring(key)
	end

	local brackets = "["..key.."]"
	return key, brackets
end

function _Comma(isLast)
	return isLast and "" or ","
end

local function _BktR(isLast)
	return _bktR .. _Comma(isLast)
end

local function _Space(space, key, isLast, noAlignLine)
	local indent
	if noAlignLine or isLast then
		indent = space .. _indent
	else
		indent = space .. "|" .. string.rep(" ", 3)
	end
	return indent
end

local function _EmptyTable(isLast)
	return _tbEmpty .. _Comma(isLast)
end

local function _Concat(...)
	return table.concat({...}, "")
end

local function _tdump(root, depthMax, excludeKey, excludeType, noAlignLine)
	if type(root) ~= "table" then
		return root
	end

	depthMax = depthMax or 3 -- 默认三层
	local cache = { [root] = "." }
	local depth = 0
	local temp = {"{"}
	local function _dump(t, space, name)
		for k,v in pairs(t) do
			local isLast = not next(t, k) --最后一个字段
			local key, keyBkt = _EscapeKey(k)

			if type(v) == "table" then
				if cache[v] then
					table.insert(temp, _Concat(space, keyBkt, _eqStr, _bktL, cache[v], _BktR(isLast)))
				else
					local new_key = name .. "." .. tostring(k)
					cache[v] = new_key .. " ->[".. tostring(v) .."]"

					-- table 深度判断
					depth = depth + 1
					if (depthMax > 0 and depth >= depthMax) or (excludeKey and excludeKey==k) then
						table.insert(temp, _Concat(space, keyBkt, _eqStr, _tbShort, _Comma(isLast)))
					else
						if next(v) then
							-- 非空table
							table.insert(temp, _Concat(space, keyBkt, _eqStr, _bktL))
							_dump(v, _Space(space, key, isLast, noAlignLine), new_key)
							table.insert(temp, _Concat(space, _BktR(isLast)))
						else
							table.insert(temp, _Concat(space, keyBkt, _eqStr, _EmptyTable(isLast)))
						end
					end
					depth = depth -1
				end
			else
				local vType = type(v)
				if not excludeType or excludeType ~= vType then
					if vType == "string" then
						v = '\"' .. v .. '\"'
					else
						v = tostring(v) or "nil"
					end
					table.insert(temp, _Concat(space, keyBkt, _eqStr, v, _Comma(isLast)))
				end
			end
		end

		--return #temp>0 and table.concat(temp,"\n") or nil
	end
	_dump(root, _indent, "")
	table.insert(temp, "}")

	return table.concat(temp, "\n")
end

local function _getcallstack(level)
	local info = debug.getinfo(level)
	if info then
		return string.format("[file]=%s,[line]=%d]: ", info.source or "?", info.currentline or 0)
	end
end

-- 树型打印一个 table,不用担心循环引用
-- depthMax 打印层数控制，默认3层（-1表示无视层数）
-- excludeKey 排除打印的key
-- excludeType 排除打印的值类型
-- noAlignLine 不打印对齐线
table.print = function(root, depthMax, excludeKey, excludeType, noAlignLine)
	if type(root) ~= "table" then
		print(root)
	else
		table.print(_tdump(root, depthMax, excludeKey, excludeType, noAlignLine))
	end
end

function xprint( ... )
	local print = _print

	local t = {_getcallstack(3)}
	local args = {...}
	local argn = select("#", ...)

	for i=1,argn do
		local value = args[i]
		local ty = type(value)
		if ty == "table" then
			table.insert(t, _tdump(value))
		elseif ty == "string" then
			table.insert(t, '\"' .. value .. '\"')
		else
			table.insert(t, tostring(value))
		end
	end

	print(table.concat(t, "\n"))
end

-- 修改默认的print，支持显示文件名和行数
local function debug_print(...)
	local prefix = _getcallstack(3)
	if prefix then
		--local tm = os.date("%Y-%m-%d %H:%M:%S", os.time())
		_print(prefix, ...)
	end
end
--print = debug_print

-- test
local cat = {
	name = "cat",
	sex = "man",
	age = 30,
	phone = {
		{type=1, number=123},
		{type=2, number=456},
	}
}
local domi = {
	name = "domi",
	sex = "man",
	age = 30,
	phone = {
		{type=1, number=1230000},
		{type=2, number=3210000},
	},
	addrbooks = {cat}
}
table.insert(domi.addrbooks, domi)

--xprint(1, 2, 3, domi)
table.print(domi, -1, nil, nil, true)