package queue

import "fmt"

var queue []int

func Push(val int) {
	queue = append(queue, val)
}
func Front() int {
	if len(queue) == 0 {
		fmt.Println("Queue is empty")
		return -1
	}
	return queue[0]
}
func Pop() {
	if len(queue) == 0 {
		fmt.Println("Queue is empty")
		return
	}
	queue = queue[1:]
}
func Empty() bool {
	return len(queue) == 0
}
