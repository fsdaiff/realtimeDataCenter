package component

import (
	"github.com/fsdaiff/logClient"
	"sync"
)

//读阻塞队列管理类
//查找阻塞队列里，受影响的部分

//查找是否相似，如果一方包含另一方
func WhetherContain(a string, b string) bool {
	len_a := len(a)
	len_b := len(b)

	var length int
	//挑选出小的
	if len_a > len_b {
		length = len_b
	} else {
		length = len_a
	}

	for i := 0; i < length; i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func createlongPollReadControl() longPollReadControl {
	var a longPollReadControl
	return a
}

type longPollReadControl struct {
	array sync.Map //map[string]*VisitorRwaitAndBuffer //监听数据
}

//add 会并发
func (wait *longPollReadControl) applyFor(input VisitorRwait) *VisitorRwaitAndBuffer {
	key := input.String()

	if value, ok := wait.array.Load(key); ok { //已存在该访问者，但可能需要更新
		value.(*VisitorRwaitAndBuffer).VisitorRwait = input
	} else { //添加新的访问者
		holmes.Debugf("添加新监听%s", key)
		wait.array.Store(key, newVisitorRwaitAndBuffer(input))
	}
	if value, ok := wait.array.Load(key); ok {
		return value.(*VisitorRwaitAndBuffer)
	} else {
		return nil
	}
}
func (wait *longPollReadControl) allKeys() []string {
	var a []string = make([]string, 0, 1000)
	wait.array.Range(func(key interface{}, value interface{}) bool {
		a = append(a, key.(string))
		return true
	})
	return a
}

//并发
func (wait *longPollReadControl) Remove(key string) {
	wait.array.Delete(key)
}

//不并发
//返回受影响的访问者,如果没有，返回长度为零
func (wait *longPollReadControl) Affect(input string) map[string]*VisitorRwaitAndBuffer {
	var a map[string]*VisitorRwaitAndBuffer = make(map[string]*VisitorRwaitAndBuffer, 0)
	wait.array.Range(func(key interface{}, value interface{}) bool {
		if WhetherContain(input, value.(*VisitorRwaitAndBuffer).GetPath().Output2String()) {
			a[key.(string)] = value.(*VisitorRwaitAndBuffer)
		}
		return true
	})
	return a
}

/*func (wait *longPollReadControl) FindTimeout() map[string]*VisitorRwaitAndBuffer {
	var a map[string]*VisitorRwaitAndBuffer = make(map[string]*VisitorRwaitAndBuffer, 0)
	now := time.Now().Unix()
	for key, value := range wait.array {
		//相关
		if now > value.Timeout {
			a[key] = value
		}
	}
	return a
}*/
