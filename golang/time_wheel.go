package main

import (
	"fmt"
	"strconv"
	"time"
)

// 定时器对象
type TimeObj struct {
	Id       uint64 // 定时对象id
	Name     string // 定时对象名字
	NeedTick uint32 // 需要几个tick（不是需要多少时间，现在定义一个tick耗时1s）
	TickOut  uint32 // 超时tick
	// callback
}

type TimeWheel struct {
	CurTick uint32      // 当前总共走过的tick数
	Len     uint32      // 该时间轮的刻度数（槽数）
	Slots   [][]TimeObj // tick数小于最大槽数的定时对象列表
	Lslots  []TimeObj   // tick数超过最大超时的定时对象列表
}

var (
	T     *TimeWheel
	Count uint64 = 0 // 用来自增做定时对象id
)

func (this *TimeWheel) Add(tick uint32) {
	Count++
	name := "t" + strconv.FormatUint(Count, 10)
	t := TimeObj{Count, name, tick, this.CurTick + tick}
	if tick > this.Len {
		this.Lslots = append(this.Lslots, t)
	} else {
		slotIdx := (this.CurTick + tick) % this.Len
		this.Slots[slotIdx] = append(this.Slots[slotIdx], t)
		fmt.Println(slotIdx)
	}

	fmt.Println("slots = ", this.Slots)
}

func (this *TimeWheel) Del(id uint64) {
	for i := 0; i < len(this.Slots); i++ {
		for index, t := range this.Slots[i] {
			if t.Id == id { // slice删除操作
				this.Slots[i] = append(this.Slots[i][:index], this.Slots[i][index+1:]...)
			}
		}
	}

	for index, t := range this.Lslots {
		if t.Id == id { // slice删除操作
			this.Lslots = append(this.Lslots[:index], this.Lslots[index+1:]...)
		}
	}
}

func (this *TimeWheel) Tick() {
	fmt.Println(time.Now())
	this.CurTick++
	curSlot := this.CurTick % this.Len
	if len(this.Slots[curSlot]) > 0 {
		for index, t := range this.Slots[curSlot] {
			if t.TickOut == this.CurTick {
				fmt.Println("time out", this.CurTick, time.Now())
				this.Slots[curSlot] = append(this.Slots[curSlot][:index], this.Slots[curSlot][index+1:]...)
			}
		}
	}

	if curSlot+1 == this.Len {
		for _, t := range this.Lslots {
			diff := t.TickOut - this.CurTick
			if diff <= this.Len {
				slot := t.TickOut % this.Len
				this.Slots[slot] = append(this.Slots[slot], t)
			}
		}
	}
}

func main() {
	fmt.Println("time wheel")
	T = new(TimeWheel)
	T.Len = 10
	T.Slots = make([][]TimeObj, 10)
	//fmt.Println(T)
	//fmt.Println(T.Slots[0])
	T.Add(10)
	T.Add(15)

	var i int32 = 0
	tick := time.NewTicker(time.Second * 1) // 1s
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			i++
			if i == 5 {
				T.Del(1)
			}

			if i == 20 {
				fmt.Println("timer end……")
				return
			}
			T.Tick()
		}
	}
}
