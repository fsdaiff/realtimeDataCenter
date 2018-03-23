package component

import (
	"encoding/json"
	"fmt"
	"logClient"
	"sync"
	"testing"
	"time"
)

var control *VisitorsControl
var defalutinput string

func init() {
	defalutinput = `{
"版本": "0.1",
"服务器类型": 0,
"标准输入采集": true,
"http日志输入采集": true,
"消息队列输入采集": false,
"文件保存行数": 1024,
"默认保存路径": "/var/log/DefaultLog",
"默认保存等级": 1,
"自动创建": true,
"实时读取支持": true,
"日志节点记录": [
{
"节点名称": "myself",
"最大保存文件数": 1024,
"保存路径": "/var/log/DefaultLog/myself",
"保存等级": 1,
"文件采集队列": [
{
"FileType": 0,
"Path": "/var/log/DefaultLog/myself.pipe"
}
]
}
]
}`
	var a interface{}
	json.Unmarshal([]byte(defalutinput), &a)
	control = NewVisitorsControl(a)
	holmes.Start()
	go control.Control()
	time.Sleep(100 * time.Millisecond)
}
func Test_controlread(t *testing.T) {
	ret := control.ReadQuickly("dgl", "日志节点记录")
	if ret.Nil() {
		t.Errorf("读错误")
	}
}
func Test_1write10read(t *testing.T) {
	var num_read int = 10
	var oneW10read = func() {
		var wg sync.WaitGroup
		var wait_start sync.WaitGroup
		var run_time chan int64 = make(chan int64, 100)
		var read_f = func(name string) {
			from := time.Now().UnixNano()
			control.RegisterNotify(name, "日志节点记录", 1)
			wait_start.Done()
			ret := control.ReadWithTime(name, "日志节点记录", 1)
			if ret.Nil() {
				t.Errorf("读失败")
			} else {
				t.Logf("%s读成功%s", name, ret)
			}
			visitor, _, timestamp := ret.Get()
			if visitor != "dgl-write" {
				t.Errorf("非预期写入者%s", visitor)
			}
			if timestamp == 0 {
				t.Errorf("时间戳错误")
			}
			wg.Done()
			to := time.Now().UnixNano()
			run_time <- (to - from)
		}
		from := time.Now().UnixNano()
		for i := 0; i < num_read; i++ {
			wg.Add(1)
			wait_start.Add(1)

			go read_f(fmt.Sprintf("dgl%d", i))
		}
		//等待全部启动
		wait_start.Wait()
		t.Logf("启动完成")
		var a interface{}
		json.Unmarshal([]byte(defalutinput), &a)
		write_start := time.Now().UnixNano()
		ret := control.WriteQuickly("dgl-write", "", a)
		if ret == nil {

		} else {
			t.Errorf("写失败:%s", ret.Error())
		}
		write_stop := time.Now().UnixNano()
		wg.Wait()
		to := time.Now().UnixNano()
		var run_time_s string
		for i := 0; i < num_read; i++ {
			select {
			case a := <-run_time:
				run_time_s += fmt.Sprintf("%d:%d\n", i+1, a)
			case <-time.After(100 * time.Microsecond):
				run_time_s += fmt.Sprintf("%d:没有收到\n", i+1)
			}
		}
		run_time_s += fmt.Sprintf("写开始时间:%d,写结束时间:%d,写运行时间%d\n", write_start, write_stop, write_stop-write_start)
		run_time_s += fmt.Sprintf("总运行时间%d\n", to-from)
		t.Log(run_time_s)
	}
	from := time.Now().UnixNano()
	for i := 0; i < 1; i++ {
		oneW10read()
	}
	to := time.Now().UnixNano()
	t.Logf("平均时间%dns", (to-from)/10)
}
