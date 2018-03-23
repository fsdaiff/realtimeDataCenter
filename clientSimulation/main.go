package main

import (
	"encoding/json"
	"logClient"
	"realtimeDataCenter/component"
	"realtimeDataCenter/socketApi"
	"sort"
	"sync/atomic"
	"time"
)

//读所有节点数据
//创建各个节点的监听路径
//写各个节点
func main() {
	defer holmes.Start(holmes.SetLevel("info")).Stop()
	var write_socket socketApi.ClientHandle
	write_socket.StartConnect(":8084", "socket write", nil)

	//读所有节点数据
	ret := write_socket.UnblockRead("")
	if !ret.Success() {
		holmes.Fatalf(ret.Reason)
	}
	var jsonValue component.JsonObject
	jsonValue.SetValue(ret.Payload)

	//创建各个节点的监听路径
	paths := component.IterAllPath("", ret.Payload)
	var read_socket socketApi.ClientHandle
	var 阻塞读计数 map[string]*int64 = make(map[string]*int64, 10000)
	var total_read int64
	read_socket.StartConnect(":8084", "socket read", func(cmd byte, payload []byte) []byte {
		if cmd != socketApi.CmdNotify {
			holmes.Fatalf("返回错误cmd%d", cmd)
		}
		var request socketApi.ChangeNotifyJson
		var ret socketApi.ReturnBase
		err := json.Unmarshal(payload, &request)
		if err != nil {
			ret.Set(false, err.Error())
		} else {
			ret.Set(true, "")
			atomic.AddInt64(阻塞读计数[request.Path], 1)
			atomic.AddInt64(&total_read, 1)
		}
		return []byte(ret.String())
	})
	for _, value := range paths {
		ret := read_socket.RegisterNotify(value, 10)
		if !ret.Success() {
			holmes.Fatalf("监听%s异常%s", value, ret.Reason)
		} else {
			阻塞读计数[value] = new(int64)
		}
	}
	holmes.Infof("监听节点数%d", len(paths))
	write_start := time.Now().UnixNano()
	//随机写各个节点
	for _, value := range paths {
		var road component.Path
		road.InputFromString(value)
		find_result := jsonValue.Find(road)
		if find_result == nil {
			holmes.Fatalf("%s节点不存在", value)
		}
		w_ret := write_socket.WriteData(value, find_result)
		if !w_ret.Success() {
			holmes.Errorf(w_ret.Reason)
		}
	}
	write_stop := time.Now().UnixNano()
	holmes.Infof("写耗费时间%d", write_stop-write_start)
	time.Sleep(100 * time.Millisecond)
	sort.Strings(paths)
	holmes.Infof("总通知量%d", total_read)
	for key, value := range paths {
		holmes.Infof("%d:%s被写入%d次", key, value, *阻塞读计数[value])
	}

}
