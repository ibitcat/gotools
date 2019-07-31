-- lua 打印table结构，参考云风的思路，稍微优化了一下
-- 加了层级判断，因为工作中有些table太多层嵌套了，导致关键信息看不到

local _print = _G.print

function _tdump(root, depthMax, excludeKey, excludeType)
	if type(root) ~= "table" then
		return root
	end

	depthMax = depthMax or 3 -- 默认三层
	local cache = { [root] = "." }
	local depth = 0
	local temp = {"{"}
	local function _dump(t, space, name)
		for k,v in pairs(t) do
			local key = tostring(k)
			if type(k) == "string" then
				key ='\"' .. tostring(k) .. '\"'
			end

			if type(v) == "table" then
				if cache[v] then
					table.insert(temp, space .. "["..key.."]" .." = " .. " {" .. cache[v].."},")
				else
					local new_key = name .. "." .. tostring(k)
					cache[v] = new_key .." ->[".. tostring(v) .."]"

					-- table 深度判断
					depth = depth + 1
					if depth>=depthMax or (excludeKey and excludeKey==k) then
						table.insert(temp,space .. "["..key.."]" .." = " .. "{ ... }")
					else
						local isLast = not next(t, k) --最后一个字段
						if next(v) then
							-- 非空table
							table.insert(temp, space .. "["..key.."]" .." = " .. "{")
							_dump(v, space .. (isLast and " " or "|") .. string.rep(" ",#key<4 and 4 or #key),new_key)
							table.insert(temp, space .. (isLast and "}" or "},"))
						else
							table.insert(temp, space .. "["..key.."]" .." = " .. (isLast and "{ }" or "{ },"))
						end
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
					local isLast = not next(t, k) --最后一个字段
					table.insert(temp, space .. "["..key.."]" .. " = " .. v ..(isLast and "" or ","))
				end
			end
		end

		--return #temp>0 and table.concat(temp,"\n") or nil
	end
	_dump(root, "    ", "")
	table.insert(temp, "}")
	return table.concat(temp, "\n")
end


-- 树型打印一个 table,不用担心循环引用
-- depthMax 打印层数控制，默认3层
-- excludeKey 排除打印的key
-- excludeType 排除打印的值类型
table.print = function(root, depthMax, excludeKey, excludeType)
	if type(root) ~= "table" then
		print(root)
	else
		table.print(_tdump(root, depthMax, excludeKey, excludeType))
	end
end

function Xprint( ... )
	local print = _print
	for i=1,arg.n do
		local value = arg[i]
		if type(value) == "table" then
			table.print_r(value)
		elseif type(value) == "string" then
			print('\"' .. value .. '\"')
		else
			print(tostring(value))
		end
	end
	print("\n")
end

-- 修改默认的print，支持显示文件名和行数
local function debug_print(...)
	local info = debug.getinfo(2)
	if info then
		--local tm = os.date("%Y-%m-%d %H:%M:%S", os.time())
		_print(string.format("[file]=%s,[line]=%d:",info.source or "?",info.currentline or 0), ...)
	end
end
print = debug_print