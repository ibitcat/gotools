package main

import (
	"fmt"
)

// 冒泡排序，冒泡排序需要排序轮数 = 参与排序元素个数 - 1
// isAsc = true表示升序
// 稳定
func BubbleSort(t []int, isAsc bool) {
	l := len(t)
	flag := true               // 如果一轮排序中，没有变化，说明已经排序好了，后面的都不用再循环了
	for i := 0; i < l-1; i++ { // 轮数循环
		for j := 0; j < l-i-1; j++ { // 每轮元素之间排序
			if isAsc {
				if t[j] > t[j+1] {
					t[j], t[j+1] = t[j+1], t[j]
					flag = false
				}
			} else {
				if t[j] < t[j+1] {
					t[j], t[j+1] = t[j+1], t[j]
					flag = false
				}
			}
		}

		if flag {
			fmt.Println("后面的已经按顺序排好了，不用再循环了")
			break
		}
	}
}

// 快速排序
// 不稳定
// 一个很好理解的思想：留出一个空位用来做交换，最后在把这个位置填回去
func QuickSort(t []int) {
	if len(t) <= 1 { // 跳出递归的条件
		return
	}

	left, right := 0, len(t)-1
	temp := t[0]
	for left < right {
		for ; right > left; right-- {
			if temp > t[right] { //交换，小的放前面去
				t[left] = t[right]
				break
			}
		}

		for ; left < right; left++ {
			if temp < t[left] { //交换，大的放后面去
				t[right] = t[left]
				break
			}
		}
		t[left] = temp
	}

	QuickSort(t[:left])
	QuickSort(t[left+1:])
}

// 直接选择排序
// 不稳定
func StraightSelectSort(t []int) {
	l := len(t)
	for i := 0; i < l-1; i++ {
		small := i
		for j := i + 1; j < l; j++ {
			if t[j] < t[small] { // 记录索引
				small = j
			}
		}

		if small != i {
			t[i], t[small] = t[small], t[i]
		}
	}
}

func main() {
	fmt.Println("hello")
	t := []int{1, 2, 5, 4, 3}
	BubbleSort(t, true)
	fmt.Println(t)

	t1 := []int{7, 2, 5, 4, 3}
	QuickSort(t1)
	fmt.Println(t1)

	t2 := []int{5, 2, 3, 7, 9, 1}
	StraightSelectSort(t2)
	fmt.Println(t2)
}
