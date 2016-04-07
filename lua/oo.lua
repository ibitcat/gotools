-- 这里是lua5.1.4 面向对象的模拟实现

module("lualib.oo", package.seeall)

local setmetatable = setmetatable
local _cls = {}
local _tnm = {}
function class(name, father)
	assert(type(name) == 'string')
	local t = _cls[name]
	if not t then
		if father then
			assert(_cls[father], father)
		end
		local cls = {}
		local meta = {
			__call = function(tlt, ...)
				assert(tlt == cls)
				local o = {}
				setmetatable(o, {__index=cls})
				o.__init(o, ...)
				return o
			end,
			__index = _cls[father] and _cls[father][1]
		}
		setmetatable(cls, meta)
		t = {cls}
		_cls[name] = t
		_tnm[cls] = name
	else
		assert(not t[2], name)
	end
	_G[name] = t[1]
	return t[1], father and _cls[father][1]
end

function single(name)		--单例
	local t = _cls[name]
	if not t then
		t = {{}, true, 0}
		_cls[name] = t
		_tnm[t[1]] = name
	else
		assert(t[2], name)
	end
	_G[name] = t[1]
	return t[1]
end

function getname(tbl)
	return self._tnm[tbl]
end

-- 例子：
--[[
oo.class("Base")			-- 基类
oo.class("Child","Base")	-- 继承


oo.single("Type")			-- 单例
]]