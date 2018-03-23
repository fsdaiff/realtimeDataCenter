package socketApi

import (
	"encoding/json"

	"fmt"

	"github.com/fsdaiff/logClient"
	"github.com/fsdaiff/realtimeDataCenter/component"
	"net"
	"sync"
)

func new_realtimeDataCenterServerSocketHandler(connection *net.TCPConn, r component.ControlRead, w component.ControlWrite) *realtimeDataCenterServerSocketHandler {
	a := new(realtimeDataCenterServerSocketHandler)
	a.ControlRead = r
	a.ControlWrite = w
	a.socketBaseApi = newRealtimeDataCenterSocket(connection, Server_flag, a.socketOutputHandle, a.closeNotify)
	a.StartProcess()
	return a
}

//一个socket所能做的事
type realtimeDataCenterServerSocketHandler struct {
	read_request_queue sync.Map //事件通知队列

	component.ControlRead
	component.ControlWrite
	socketBaseApi
	退出标志位 bool //true时所开启的子协程退出
}

//读socket数据处理处理
func (socketHandler *realtimeDataCenterServerSocketHandler) socketOutputHandle(cmd byte, payload []byte) []byte {
	//客户端请求
	//1、非阻塞读
	//2、写数据
	//3、添加监听数据
	switch cmd {
	case CmdUnblockRead:
		var request visitorBase
		var ret ReturnWithPayload
		err := json.Unmarshal(payload, &request)
		if err == nil {
			result := socketHandler.ReadQuickly(request.Visitor, request.Path)
			if result.Nil() {
				ret.Set(false, "未找到该路径"+request.Path)
			} else {
				ret.Set(true, "")
				ret.Payload = result
			}
		} else {
			ret.Set(false, err.Error())
		}
		return []byte(ret.String())
	case CmdWrite:
		var request visitorWrite
		var ret ReturnBase
		err := json.Unmarshal(payload, &request)
		if err == nil {
			err := socketHandler.WriteQuickly(request.Visitor, request.Path, request.Data)
			if err != nil {
				ret.Set(false, err.Error())
			} else {
				ret.Set(true, "")
			}
		} else {
			ret.Set(false, err.Error())
		}
		return []byte(ret.String())
	case CmdRegisterNotify:
		var request visitorRequestNotify
		var ret ReturnWithPayload
		err := json.Unmarshal(payload, &request)
		if err == nil {
			result := socketHandler.ReadQuickly(request.Visitor, request.Path)
			if result.Nil() {
				ret.Set(false, "未找到该路径"+request.Path)
			} else {
				//注册
				register_result := socketHandler.RegisterNotify(request.Visitor, request.Path, request.Timeout)
				if register_result.Nil() {
					ret.Set(false, "未找到该路径"+request.Path)
				} else {
					//数据节点存在
					ret.Set(true, "")
					ret.Payload = result
					//开启监听线程
					var 等待线程启动完成 sync.WaitGroup
					等待线程启动完成.Add(1)
					go socketHandler.readPoll(request.Visitor, request.Path, request.Timeout, &等待线程启动完成)
					等待线程启动完成.Wait()
				}
			}
		} else {
			ret.Set(false, err.Error())
		}
		return []byte(ret.String())
	case CmdNotify: //不存在

	case CmdHeart: //心跳
	}
	var ret ReturnBase
	ret.Set(false, fmt.Sprintf("请求的指令不存在%d", cmd))
	return []byte(ret.String())
}

//读监听循环，用于在客户端发送监听请求后，开始对某路径的监听，有监听事件时将数据发送到socket上
func (socketHandler *realtimeDataCenterServerSocketHandler) readPoll(visitor_name string, road string, timeout int, wait_start *sync.WaitGroup) {
	var routinue_id string = fmt.Sprintf("[%s]协程[%s:%s-	%d]--", socketHandler.GetId(), visitor_name, road, timeout)
	defer holmes.Infof("%s退出", routinue_id)
	var retJson changeNotify
	holmes.Debugf("%s启动", routinue_id)
	wait_start.Done()
	for {
		//等待数据改变
		//注意该监听此时可能未注册成功
		retJson.SetPayload(road, socketHandler.ReadBuffer(visitor_name, road, timeout))
		//返回数据格式保护
		if retJson.Payload.Nil() {
			holmes.Errorf("%sReadWithTime返回的数据为空，不发送到客户端!", routinue_id)
			continue
		}
		//发送

		ret, err := socketHandler.Request(CmdNotify, []byte(retJson.String()))
		if err != nil {
			holmes.Errorln(err)
		} else {
			var result ReturnBase
			err = json.Unmarshal(ret, &result)
			if err != nil {
				holmes.Errorln(routinue_id, err)
			} else {
				if !result.Success() {
					holmes.Errorf("%s:%s", routinue_id, result.Reason)
				}
			}
		}

		//是否退出
		if socketHandler.退出标志位 {
			return
		}
	}
}
func (socketHandler *realtimeDataCenterServerSocketHandler) closeNotify(reason string) {
	socketHandler.退出标志位 = true
}

//监听处理程序
type ServerListenHandle struct {
	listenAddr *net.TCPAddr
	component.ControlRead
	component.ControlWrite
}

func (server *ServerListenHandle) StartListen(addr string, r component.ControlRead, w component.ControlWrite) error {
	server.ControlRead = r
	server.ControlWrite = w
	var err error
	//转换为net.TCPAddr格式
	server.listenAddr, err = net.ResolveTCPAddr("tcp4", addr)
	if err != nil {
		return err
	}
	listens, err := net.ListenTCP("tcp4", server.listenAddr)
	if err != nil {
		if listens != nil {
			listens.Close()
		}
		return err
	}
	holmes.Eventf("实时数据汇总进程建立socketApi监听%s", listens.Addr())
	for {
		connection, err := listens.AcceptTCP()
		if err == nil {
			var handler *realtimeDataCenterServerSocketHandler = new_realtimeDataCenterServerSocketHandler(connection, r, w)
			holmes.Debugf("%s连接建立", handler.GetId())
		} else {
			holmes.Errorf("%s:%s", server.listenAddr, err)
		}
	}

}
