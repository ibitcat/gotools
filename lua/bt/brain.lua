-- 大脑（ai的封装）

--oo.class("Brain","AIBTree")
oo.class("Brain")
function Brain:__init(owner)
	assert(owner)
	--local cnf = owner:getConfig()
	--AIBTree.__init(self, owner, cnf.ai, cnf)

	self._stopped = true			--大脑是否停止(默认停止)
	self._bt = nil					--大脑的行为树
	self._thinkTimerId = nil		--大脑思考的定时器
	self._lastThinkTime = 0			--大脑上一次思考的时间戳（ms）
	--self._blackboard = {}			--黑板
	--self._owner = owner			--大脑的拥有者
end

-- 行为树强制update一次
function Brain:forceUpdate()
	if self._stopped then return end
	if self._bt then
		self._bt:forceUpdate()
	end
	
	if self._thinkTimerId then
		self._owner:removeTimerListener(self._thinkTimerId)
		self._thinkTimerId = nil
	end

	--强制思考一次
	print("强制思考一次")
	self:think()
end

function Brain:getSleepTime()
	if self._bt then
		return self._bt:getSleepTime()
	end
	
	return 0
end

-- 创建怪物后并给怪物创建一个大脑，并启动大脑
function Brain:start()
	--创建行为树bt，由具体的子类重写(必须要实现该方法)
	self:onStart()

	--行为树的初始化，由具体的子类重写
	if self.onInit then
		self:onInit()
	end

	-- 触发大脑思考的事件注册
	self:addEventHandler()

	-- wake up
	-- 如果是主动怪，立即开启思考
	local thinkType = true --测试用，都是主动怪
	if thinkType and self._stopped then
		assert(self._bt)
		self._thinkTimerId = self._owner:addTimerListener(self, "think", math.random(100,200))
		self._stopped = false
	end
end

-- 事件注册
function Brain:addEventHandler()
	assert(self._bt)

	-- 大脑触发思考事件
	self._owner:addEventListener(self, EVENT.BE_ATTACKED_END, 'onEvent')
	-- 其他触发思考的事件注册
	-- TODO

	-- 大脑停止思考事件
	self._owner:addEventListener(self, EVENT.BE_KILLED, "stop")
end

-- 停止后，删除所有的事件
function Brain:delEventHandler()
	self._owner:removeEventListener(self)
end

-- 事件触发大脑思考，比如：被攻击等
function Brain:onEvent()
	if self._stopped or not self._thinkTimerId then
		self._stopped = false
		self:think()
	end
end

--  update
function Brain:update()
	if self.doUpdate then
		self:doUpdate()
	end

	if self._bt then
		self._bt:update()
	end

	if self.onUpdate then
		self:onUpdate()
	end
end

-- 大脑停止思考
function Brain:stop()
	if self._bt then
		self._bt:stop()
	end
	self._stopped = true
	self._lastThinkTime = 0
	self:delEventHandler()

	if self._thinkTimerId then
		self._owner:removeTimerListener(self._thinkTimerId)
		self._thinkTimerId = nil
	end

	if self.onStop then
		self:onStop()
	end 
end

-- 大脑开始思考一次
function Brain:think()
	self._thinkTimerId = nil --重置定时器

	-- 大脑没有拥有者 
	-- 或者 拥有者未出生 
	-- 或者 拥有者死亡，则停止思考
	if not self._owner
		or not self._owner:isValid()
		or self._owner:isDead() then
		return
	end

	self._decitionVersion = self._decitionVersion + 1
	self:update() --大脑思考一次
	self._lastThinkTime = env.unixtimeMs()

	if self._stopped then return end
	local sleep_amount = self:getSleepTime()
	if sleep_amount then
		sleep_amount = sleep_amount>=10 and sleep_amount or 10 --必须是10ms的倍数
		--print("sleep_amount------------>>",sleep_amount)
		local tick = math.floor(sleep_amount/10)
		self._thinkTimerId = self._owner:addTimerListener(self, "think", tick)
	else
		--休眠
		log("大脑休眠")
	end
end

function Brain:btString()
	if self._bt then
		local btStr = self._bt._root:getTreeString()
		print(btStr or "empty btree")
	end
end