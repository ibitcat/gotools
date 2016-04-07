-- 为了递归打开lua文件，并且在lua文件的最顶行插入 '-- $Id$'
-- 插入的这行注释，是为了svn提交的时候，自动填充提交人id和提交时间的

require "lfs"

function addId(f)
	local linestab = {}
	local fl = io.open(f)
	for line in fl:lines() do
		table.insert(linestab,line)
		--print (line)
	end
	fl:close()

	table.insert(linestab,1,"-- $Id$")
	fl = io.open(f,"w")
	for _,line in ipairs(linestab) do
		fl:write(line..'\n')
	end
	fl:close()
end


function findDir(pa)
	for file in lfs.dir(pa) do
		if file ~= "." and file ~= ".." then
	    	local f = pa..'\\'..file
			if string.find(f,"%.lua") ~= nil then
				addId(f)
			end

		local attr = lfs.attributes(f)
	    	assert (type(attr) == "table")
	    	if attr.mode == "directory" then
	        	findDir(f)
			end
		end
	end
end

local currentFolder = [[d:\map]]
findDir(currentFolder)