package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"logClient"
	"logServer/config"
	"os"
	"realtimeDataCenter/resetfulClient"
	"realtimeDataCenter/socketApi"
	"strconv"
	"sync"
	//"testing"
	"time"
)

func cycleRead(name string) {
	dataCenterClient.SetvisitorName(name)
	ret := dataCenterClient.HttpReadWithTimeout("", 10, []config4log.LogNode{})
	if ret.Success() {
	} else {
		holmes.Errorf("通信失败:%s", ret.Reason)
	}
}

//模拟长轮询读
func cycleReads(visitor_num int) {
	/*var cycle = func(name string) {
		for {
			cycleRead(name)
		}
	}*/
	var wait_register sync.WaitGroup
	for i := 0; i < visitor_num; i++ {
		//go cycle(fmt.Sprintf("dgl%d", i))
		wait_register.Add(1)
		go socketCycleRead(i, &wait_register)
	}
	wait_register.Wait()
}
func socketCycleRead(visitor_num int, wait_register *sync.WaitGroup) {
	defer wait_register.Done()
	var change_notify = func(cmd byte, payload []byte) []byte {
		var ret socketApi.ReturnBase
		if cmd != socketApi.CmdNotify {
			ret_str := fmt.Sprintf("客户端接收到没有预计到指令%d.[%s]", cmd, string(payload))
			holmes.Errorf(ret_str)
			ret.Set(false, ret_str)
		} else {
			var a socketApi.ChangeNotifyJson
			err := json.Unmarshal(payload, &a)
			if err != nil {
				holmes.Errorf("%s", err.Error())
				ret.Set(false, err.Error())
			} else {
				//holmes.Infof()("收到改变通知%s", string(payload))
				ret.Set(true, "")
				if a.Timestamp == 0 {
					holmes.Errorf("时间戳为零")
				}
			}

		}
		return []byte(ret.String())
	}
	var client_socket socketApi.ClientHandle
	err := client_socket.StartConnect(":8084", fmt.Sprintf("socket-dgl%d", visitor_num), change_notify)
	if err != nil {
		holmes.Errorf("%s", err.Error())
		return
	}
	//defer client_socket.CLoseSocket()
	register_ret := client_socket.RegisterNotify("", 10)

	if register_ret.Success() {
	} else {
		holmes.Errorf("注册监听失败%s", register_ret.Reason)
		return
	}
}

var standard string

func indent2string(a interface{}) string {
	datas, _ := json.Marshal(a)
	var out bytes.Buffer
	json.Indent(&out, datas, "", "\t")
	return out.String()
}

func init() {
	//创建配置文件

	/*input := `{
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

	var a config4log.Config
	json.Unmarshal([]byte(input), &a)*/
	var x int = 4
	var y int = 3
	a := CreateStdBtree(x, 0, y)
	var i, j int
	bianli7(a, &i)
	holmes.Infof("所有节点数:%d", i)
	bianli8(a, &j)
	holmes.Infof("所有叶节点数:%d", j)
	standard = indent2string(a)

	var fileName = "config.json"
	os.Remove(fileName)
	fd, _ := os.Create(fileName)
	fd.Write([]byte(standard))
	fd.Sync()
	fd.Close()
	//dataCenterClient.SetServerAddr("http://127.0.0.1:8083/v1.0/realtime")
	go main()
	//等待初始化完成
	time.Sleep(1000 * time.Millisecond)

	var num int = 100
	cycleReads(num)
	holmes.Infof("长轮询读请求者数量%d", num)
	//time.Sleep(2 * time.Second)

	//socketApi client
}

////////////////////////////////////////////////
//组建一个对称的b树
func CreateStdBtree(elements_num int, deep_position int, max_deep int) map[string]interface{} {
	a := make(map[string]interface{})

	//创建子节点，合并子节点
	if deep_position < max_deep {
		for i := 0; i < elements_num; i++ {
			child_n := CreateStdBtree(elements_num, deep_position+1, max_deep)
			a["容器:层"+strconv.Itoa(deep_position)+"-序号"+strconv.Itoa(i)] = child_n
		}
		return a
	} else {
		//叶子节点
		b := make(map[string]interface{})

		for i := 0; i < elements_num; i++ {
			b["叶子"+strconv.Itoa(deep_position)+"-"+strconv.Itoa(i)] = i
		}
		return b
	}
}
func CreateBtreeRoad2leaf(max_deep int) string {
	var str string
	for i := 0; i < max_deep; i++ {
		str += "容器:层" + strconv.Itoa(i) + "-序号0/"
	}
	return str
}
func bianli7(data map[string]interface{}, father *int) {
	for _, val := range data {
		switch val.(type) {
		case map[string]interface{}: //容器节点
			(*father) += 1
			bianli7(val.(map[string]interface{}), father)
		default: //叶子节点
			(*father) += 1
		}
	}
}
func bianli8(data map[string]interface{}, father *int) {
	for _, val := range data {
		switch val.(type) {
		case map[string]interface{}: //容器节点
			bianli8(val.(map[string]interface{}), father)
		default: //叶子节点
			(*father) += 1
		}
	}
}

///////////////////////////////////////////////

/*
func Test_read(t *testing.T) {
	t.Logf("读所有")

	ret := dataCenterClient.HttpRead("", &config4log.Config{})
	if ret.Success() {
		output := indent2string(ret.Payload)
		if output != standard {
			t.Errorf("读所有不正确\n%s\n%s", standard, output)
		} else {
			t.Logf("通信成功，载荷:%v", ret.Payload.(*config4log.Config).Version)
		}
	} else {
		t.Errorf("通信失败:%s", ret.Reason)
	}
}
func Test_writeAndRead(t *testing.T) {
	t.Logf("阻塞读然后写")
	var over chan bool = make(chan bool, 0) //
	go func() {
		ret := dataCenterClient.HttpReadWithTimeout("", 1, &config4log.Config{})
		if ret.Success() {
			output := indent2string(ret.Payload)
			if output != standard {
				t.Errorf("读所有不正确\n%s\n%s", standard, output)
			} else {
				t.Logf("通信成功，载荷:%v", ret.Payload.(*config4log.Config).Version)
			}
		} else {
			t.Errorf("通信失败:%s", ret.Reason)
		}
		over <- true
	}()
	time.Sleep(100 * time.Microsecond)
	ret := dataCenterClient.HttpWrite("", []byte(standard))
	if ret.Success() {
		t.Logf("写成功")
	} else {
		t.Logf("写失败:%v", ret)
	}
	select {
	case <-over:
	case <-time.After(time.Millisecond * 10):
		t.Error("等待阻塞式读超时")
	}
}

func Test_10read1Write(b *testing.T) {
	var num_read int = 10
	var oneW10read = func() {
		var wg sync.WaitGroup
		var wait_start sync.WaitGroup
		var run_time chan int64 = make(chan int64, 100)
		var read_f = func(visitorName string) {
			from := time.Now().UnixNano()
			wait_start.Done()
			dataCenterClient.SetvisitorName(visitorName)
			ret := dataCenterClient.HttpReadWithTimeout("日志节点记录", 1, []config4log.LogNode{})
			if ret.Success() {
			} else {
				b.Errorf("通信失败:%s", ret.Reason)
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
		b.Logf("启动完成")
		time.Sleep(time.Millisecond * 2)
		write_start := time.Now().UnixNano()
		ret := dataCenterClient.HttpWrite("", []byte(standard))
		if ret.Success() {

		} else {
			b.Errorf("写失败:%v", ret)
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
		run_time_s += fmt.Sprintf("写运行时间%d\n", write_stop-write_start)
		run_time_s += fmt.Sprintf("总运行时间%d\n", to-from)
		b.Log(run_time_s)
	}
	from := time.Now().UnixNano()
	for i := 0; i < 1; i++ {
		oneW10read()
	}
	to := time.Now().UnixNano()
	b.Logf("平均时间%dns", (to-from)/10)
}
func Test_socket_write(t *testing.T) {
	var client_socket socketApi.ClientHandle
	err := client_socket.StartConnect(":8084", "socket-dgl1", nil)
	if err != nil {
		t.Errorf("%s", err.Error())
		return
	}
	defer client_socket.CLoseSocket("Test_socket_write over")
	var a interface{}
	json.Unmarshal([]byte(standard), &a)
	ret := client_socket.WriteData("", a)
	if ret.Success() {
		t.Logf("写成功")
	} else {
		t.Errorf("写失败:%s", ret.Reason)
	}
}
func Test_socket_read(t *testing.T) {
	t.Logf("读所有")
	var client_socket socketApi.ClientHandle
	err := client_socket.StartConnect(":8084", "socket-dgl1", nil)
	if err != nil {
		t.Errorf("%s", err.Error())
		return
	}
	defer client_socket.CLoseSocket("Test_socket_read over")
	ret := client_socket.UnblockRead("")
	if ret.Success() {
		output := indent2string(ret.Payload)
		if len(output) != len(standard) {
			t.Errorf("读所有不正确\n%s\n%s", standard, output)
		} else {
			t.Logf("通信成功，载荷:%s", output)
		}
	} else {
		t.Errorf("通信失败:%s", ret.Reason)
	}
}
func Test_socket_1write1read(t *testing.T) {
	t.Logf("一写一读所有")
	var wait_notify sync.WaitGroup
	var change_notify = func(cmd byte, payload []byte) []byte {
		defer wait_notify.Done()
		var ret socketApi.ReturnBase
		if cmd != socketApi.CmdNotify {
			ret_str := fmt.Sprintf("客户端接收到没有预计到指令%d.[%s]", cmd, string(payload))
			t.Errorf(ret_str)
			ret.Set(false, ret_str)
		} else {
			var a socketApi.ChangeNotifyJson
			err := json.Unmarshal(payload, &a)
			if err != nil {
				t.Errorf("%s", err.Error())
				ret.Set(false, err.Error())
			} else {
				t.Logf("收到改变通知%s", string(payload))
				ret.Set(true, "")
			}

		}
		return []byte(ret.String())
	}
	var client_socket socketApi.ClientHandle
	err := client_socket.StartConnect(":8084", "socket-dgl1", change_notify)
	if err != nil {
		t.Errorf("%s", err.Error())
		return
	}
	defer client_socket.CLoseSocket("Test_socket_1write1read over")
	register_ret := client_socket.RegisterNotify("", 1)
	wait_notify.Add(1)

	if register_ret.Success() {
		t.Logf("注册监听成功")
	} else {
		t.Errorf("注册监听失败%s", register_ret.Reason)
		return
	}

	var a interface{}
	json.Unmarshal([]byte(standard), &a)
	ret := client_socket.WriteData("", a)
	if ret.Success() {
		t.Logf("写成功")
	} else {
		t.Errorf("写失败:%s", ret.Reason)
	}
	//time.Sleep(1 * time.Second)
	wait_notify.Wait()
}*/
