package main

import (
	//"encoding/json"
	"flag"
	"github.com/fsdaiff/logClient"
	"github.com/fsdaiff/realtimeDataCenter/component"
	"github.com/fsdaiff/realtimeDataCenter/restfulApi"
	"github.com/fsdaiff/realtimeDataCenter/socketApi"
	"github.com/json-iterator/go"
	"io/ioutil"
	"os"
	"time"
)

var control *component.VisitorsControl

func main() {
	var config_path *string = flag.String("config", "config.json", "数据文件路径")
	var http_addr *string = flag.String("httpport", ":8083", "restfulApi监听地址")
	var socket_addr *string = flag.String("scoketport", ":8084", "scoketApi监听地址")
	var level *string = flag.String("level", "info", "设置日志输出等级 debug/info/warn/error")
	flag.Parse()
	defer holmes.Start(holmes.SetLevel(*level), holmes.SetDistributeMode("实时数据汇总节点", "127.0.0.1:8082", "/var/log/DefaultLog/realtimeCenter", "INFO")).Stop()
	defer holmes.Infof("系统退出\n")
	holmes.Infof("数据文件路径%s\n", *config_path)
	holmes.Infof("restfulApi监听地址%s\n", *http_addr)
	holmes.Infof("scoketApi监听地址%s\n", *socket_addr)
	holmes.Infof("日志输出最低等级%s\n", *level)
	fd, err := os.Open(*config_path)
	if err != nil {
		holmes.Fatalln(err)
	}
	datas, err := ioutil.ReadAll(fd)
	if err != nil {
		holmes.Fatalln(err)
	}
	start := time.Now().UnixNano()
	var input interface{}
	err = jsoniter.Unmarshal(datas, &input)
	if err != nil {
		holmes.Fatalln(err)
	} else {
		holmes.Infof("数据结构:\n%s", string(datas))
		end := time.Now().UnixNano()
		holmes.Infof("解码时间:%dns", end-start)
	}

	control = component.NewVisitorsControl(input)
	go control.Control()
	defer holmes.Infoln(control.PerformanceShow())

	var restful rtCenterApi.Restful
	go restful.Register(*http_addr, control, control)

	var socket_api socketApi.ServerListenHandle
	err = socket_api.StartListen(*socket_addr, control, control)
	if err != nil {
		holmes.Errorf("socketApi监听错误%s", err.Error())
	}
}
