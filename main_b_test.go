package main

import (
	"encoding/json"
	"logClient"
	"logServer/config"
	"realtimeDataCenter/resetfulClient"
	"realtimeDataCenter/socketApi"
	"testing"
	//"time"
)

func Benchmark_socketwrite(b *testing.B) {
	var client_socket socketApi.ClientHandle
	err := client_socket.StartConnect(":8084", "socket-dgl1-write", nil)
	if err != nil {
		b.Errorf("%s", err.Error())
		return
	}
	defer client_socket.CLoseSocket("Benchmark_socketwrite测试完成退出")
	var a interface{}
	json.Unmarshal([]byte(standard), &a)

	//var r ControlRead = control
	b.StopTimer() //调用该函数停止压力测试的时间计数

	//做一些初始化的工作,例如读取文件数据,数据库连接之类的,
	//这样这些时间不影响我们测试函数本身的性能

	b.StartTimer() //重新开始时间
	b.ReportAllocs()
	b.SetParallelism(1)
	for i := 0; i < b.N; i++ {
		//oneW10read()
		ret := client_socket.WriteData("", a)
		if ret.Success() {

		} else {
			b.Errorf("写失败:%s", ret.Reason)
		}
		//next <- i
		//r.ReadQuickly("read", "data/config_list/reg_conf/s")

	}
	//dataCenterClient.HttpWrite("", []byte(standard))
	//next <- false
}

//写影响读速度测试
/*func Benchmark_writeAffectRead(b *testing.B) {
	//var r ControlRead = control
	b.StopTimer() //调用该函数停止压力测试的时间计数

	//做一些初始化的工作,例如读取文件数据,数据库连接之类的,
	//这样这些时间不影响我们测试函数本身的性能

	b.StartTimer() //重新开始时间
	b.ReportAllocs()
	b.SetParallelism(1)
	for i := 0; i < b.N; i++ {
		//oneW10read()
		dataCenterClient.HttpWrite("", []byte(standard))
		time.Sleep(time.Microsecond * 2)
		//next <- i
		//r.ReadQuickly("read", "data/config_list/reg_conf/s")

	}
	//dataCenterClient.HttpWrite("", []byte(standard))
	//next <- false
}*/

//读速度测试
func Benchmark_read(b *testing.B) {
	//var r ControlRead = control

	b.StopTimer() //调用该函数停止压力测试的时间计数

	//做一些初始化的工作,例如读取文件数据,数据库连接之类的,
	//这样这些时间不影响我们测试函数本身的性能

	b.StartTimer() //重新开始时间
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		dataCenterClient.HttpRead("", &config4log.Config{})
		//r.ReadQuickly("read", "data/config_list/reg_conf/s")
	}
}

//json编码解码所占时间
func Benchmark_json(b *testing.B) {
	//var r ControlRead = control
	b.StopTimer() //调用该函数停止压力测试的时间计数

	//做一些初始化的工作,例如读取文件数据,数据库连接之类的,
	//这样这些时间不影响我们测试函数本身的性能

	b.StartTimer() //重新开始时间
	b.ReportAllocs()
	b.SetParallelism(1)
	for i := 0; i < b.N; i++ {
		var a config4log.Config
		err := json.Unmarshal([]byte(standard), &a)
		if err != nil {
			b.Error(err)
		}
		_, err = json.Marshal(a)
		if err != nil {
			b.Error(err)
		}

	}
	//dataCenterClient.HttpWrite("", []byte(standard))
	//next <- false
}

//写速度测试
/*func Benchmark_write(b *testing.B) {
	//var r ControlRead = control

	b.StopTimer() //调用该函数停止压力测试的时间计数

	//做一些初始化的工作,例如读取文件数据,数据库连接之类的,
	//这样这些时间不影响我们测试函数本身的性能

	b.StartTimer() //重新开始时间
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		dataCenterClient.HttpWrite("", []byte(standard))
		//r.ReadQuickly("read", "data/config_list/reg_conf/s")
	}
}*/
