package revel

import (
    "sync"
    "fmt"
)

type (
	SimpleLockStack struct {
		Current  *SimpleLockStackElement
		Creator  func() interface{}
		len      int
		capacity int
		active   int
		lock     sync.Mutex
	}
	SimpleLockStackElement struct {
		Value    interface{}
		Previous *SimpleLockStackElement
		Next     *SimpleLockStackElement
	}
	ObjectDestroy interface {
		Destroy()
	}
)

func NewStackLock(size int, creator func() interface{}) *SimpleLockStack {
	ss := &SimpleLockStack{lock: sync.Mutex{}, Current: &SimpleLockStackElement{Value: creator()}, Creator: creator}
	if size > 0 {
		elements := make([]SimpleLockStackElement, size-1)
		current := ss.Current
		for i := range elements {
			e := elements[i]
			if creator != nil {
				e.Value = creator()
			}
			current.Next = &e
			e.Previous = current
			current = &e
		}
		ss.capacity, ss.len, ss.active = size, size, 0

		ss.Current = current

		//nc := *current
		//for i:=0;nc.Previous!=nil; {
		//    println("index",i,nc.Value,nc.Previous,nc.Next)
		//    nc=*nc.Previous
		//}
		//println("index",nc.Value,nc.Next)
	}
	return ss
}
func (s *SimpleLockStack) Pop() (value interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.len == 0 {
		// Pool is empty, create a new item to return
		if s.Creator != nil {
			value = s.Creator()
		}
	} else {
		value = s.Current.Value
		s.len--
		if s.Current.Previous != nil {
			s.Current = s.Current.Previous
		}
	}
	// println("Pop ",value, s.len, s.active, s.capacity, s.Current.Next)
    s.active++
	return
}
func (s *SimpleLockStack) Push(value interface{}) {
	if d, ok := value.(ObjectDestroy); ok {
		d.Destroy()
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.len == 0 {
		s.Current.Value = value
	} else {
		if s.Current.Next == nil {
			s.Current.Next = &SimpleLockStackElement{Value: value, Previous: s.Current}
			s.capacity++
		} else {
			s.Current.Next.Value = value
		}
		s.Current = s.Current.Next
	}
	s.len++
    s.active--
	//println("Push ",value, s.len, s.active, s.capacity)
	return
}
func (s *SimpleLockStack) Len() int {
	return s.len
}
func (s *SimpleLockStack) Capacity() int {
	return s.capacity
}
func (s *SimpleLockStack) Active() int {
	return s.active
}
func (s *SimpleLockStack) String() string {
    return fmt.Sprintf("SS: Capacity:%d Active:%d Stored:%d",s.capacity,s.active,s.len)
}
