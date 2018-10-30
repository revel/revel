package utils

import (
	"testing"
)

type SimpleStackTest struct {
	index int
}

func TestUnique(b *testing.T) {
	stack := NewStackLock(10, 40, func() interface{} {
		newone := &SimpleStackTest{}
		return newone
	})
	values := []interface{}{}
	for x := 0; x < 10; x++ {
		values = append(values, stack.Pop())
	}
	if stack.active != 10 {
		b.Errorf("Failed to match 10 active %v ", stack.active)
	}
	value1 := stack.Pop().(*SimpleStackTest)
	value1.index = stack.active
	value2 := stack.Pop().(*SimpleStackTest)
	value2.index = stack.active
	value3 := stack.Pop().(*SimpleStackTest)
	value3.index = stack.active

	if !isDifferent(value1, value2, value3) {
		b.Errorf("Failed to get unique values")
	}

	if stack.active != 13 {
		b.Errorf("Failed to match 13 active %v ", stack.active)
	}

	for _, v := range values {
		stack.Push(v)
	}
	if stack.len != 10 {
		b.Errorf("Failed to match 10 len %v ", stack.len)
	}
	if stack.capacity != 10 {
		b.Errorf("Failed to capacity 10 len %v ", stack.capacity)
	}

	stack.Push(value1)
	stack.Push(value2)
	stack.Push(value3)
	if stack.capacity != 13 {
		b.Errorf("Failed to capacity 13 len %v ", stack.capacity)
	}

	value1 = stack.Pop().(*SimpleStackTest)
	value2 = stack.Pop().(*SimpleStackTest)
	value3 = stack.Pop().(*SimpleStackTest)
	println(value1, value2, value3)
	if !isDifferent(value1, value2, value3) {
		b.Errorf("Failed to get unique values")
	}

}
func TestLimits(b *testing.T) {
	stack := NewStackLock(10, 20, func() interface{} {
		newone := &SimpleStackTest{}
		return newone
	})
	values := []interface{}{}
	for x := 0; x < 50; x++ {
		values = append(values, stack.Pop())
	}
	if stack.active != 50 {
		b.Errorf("Failed to match 50 active %v ", stack.active)
	}
	for _, v := range values {
		stack.Push(v)
	}
	if stack.Capacity() != 20 {
		b.Errorf("Failed to match 20 capcity %v ", stack.Capacity())
	}

}
func isDifferent(values ...*SimpleStackTest) bool {
	if len(values) == 2 {
		return values[0] != values[1]
	}
	for _, v := range values[1:] {
		if values[0] == v {
			return false
		}
	}
	return isDifferent(values[1:]...)
}

func BenchmarkCreateWrite(b *testing.B) {
	stack := NewStackLock(0, 40, func() interface{} { return &SimpleStackTest{} })
	for x := 0; x < b.N; x++ {
		stack.Push(x)
	}
}
func BenchmarkAllocWrite(b *testing.B) {
	stack := NewStackLock(b.N, b.N+100, func() interface{} { return &SimpleStackTest{} })
	for x := 0; x < b.N; x++ {
		stack.Push(x)
	}
}
func BenchmarkCreate(b *testing.B) {
	NewStackLock(b.N, b.N+100, func() interface{} { return &SimpleStackTest{} })
}
func BenchmarkParrallel(b *testing.B) {
	stack := NewStackLock(b.N, b.N+100, func() interface{} { return &SimpleStackTest{} })
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for x := 0; x < 50000; x++ {
				stack.Push(x)
			}
		}
	})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for x := 0; x < 50000; x++ {
				stack.Pop()
			}
		}
	})
}
