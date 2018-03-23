package socketApi

import (
	"fmt"
	"github.com/fsdaiff/realtimeDataCenter/component"
)

//基础访问格式
type visitorBase struct {
	Path    string `json:"path"`
	Visitor string `json:"visitor"`
}

func (ret visitorBase) String() string {
	return fmt.Sprintf(`{"path":"%s","visitor":"%s"}`, ret.Path, ret.Visitor)
}

//写访问者
type visitorWrite struct {
	visitorBase
	Data interface{} `json:"data"`
}
type ChangeNotifyJson struct {
	visitorWrite
	Timestamp int64 `json:"timestamp"`
}

//
type visitorRequestNotify struct {
	visitorBase
	Timeout int `json:"timeout"`
}

func (ret visitorRequestNotify) String() string {
	return fmt.Sprintf(`{"path":"%s","visitor":"%s","timeout":%d}`, ret.Path, ret.Visitor, ret.Timeout)
}

//读通知
type changeNotify struct {
	visitorBase
	timestamp int64
	Payload   component.ReadResult
}

func (ret *changeNotify) SetPayload(path string, result component.ReadBufferResult) {
	ret.Path = path
	ret.Visitor, ret.Payload, ret.timestamp = result.Get()

}
func (ret changeNotify) String() string {
	if ret.Payload.Nil() {
		ret.Payload = "null"
	}
	return fmt.Sprintf(`{"path":"%s","visitor":"%s","timestamp":%d,"data":%s}`, ret.Path, ret.Visitor, ret.timestamp, ret.Payload)
}

//基础返回格式
type ReturnBase struct {
	Status string `json:"status"` //ok
	Reason string `json:"reason"`
}

func (ret ReturnBase) Success() bool {
	if ret.Status == "ok" {
		return true
	}
	return false
}

//成功:true 失败:false
func (ret *ReturnBase) Set(success bool, fail_reason string) {
	if success {
		ret.Status = "ok"
	} else {
		ret.Status = "fail"
		ret.Reason = fail_reason
	}
}
func (ret ReturnBase) String() string {
	//值检查
	if ret.Status != "ok" && ret.Status != "fail" {
		ret.Status = "fail"
		ret.Reason = "ReturnBase未初始化"
	}
	return fmt.Sprintf(`{"status":"%s","reason":"%s"}`, ret.Status, ret.Reason)
}

type ReturnWithPayload struct {
	ReturnBase
	Payload component.ReadResult `json:"data"`
}

func (ret ReturnWithPayload) String() string {
	if ret.Payload.Nil() {
		ret.Payload = "null"
	}
	return fmt.Sprintf(`{"status":"%s","reason":"%s","data":%s}`, ret.Status, ret.Reason, ret.Payload)
}

type ReturnWithJsonPayload struct {
	ReturnBase
	Payload interface{} `json:"data"`
}

type retFromSocket struct {
	data []byte
	err  error
}
type ret2Socket struct {
	data  []byte
	cmd   byte //指令
	order uint64
}
type requestFromSocket struct {
	order   uint64             //序号
	cmd     byte               //指令
	payload []byte             //请求数据
	ret     chan retFromSocket //返回数据
}
