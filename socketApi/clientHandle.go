package socketApi

import (
	"encoding/json"
	"fmt"
	"github.com/fsdaiff/logClient"
	"github.com/fsdaiff/realtimeDataCenter/component"
	"net"
	"sync"
	"time"
)

const DefaultServerAddr = "127.0.0.1:8084"

//数据改变通知回调
//连接断开回调

type ClientHandle struct {
	serverAddr    *net.TCPAddr
	visitorName   string
	autoReconnect bool //是否掉线重连
	子协程退出标识符      bool
	socketBaseApi
	子协程退出等待        sync.WaitGroup
	server_request SocketOutputFunc //回调队列
	connection     *net.TCPConn
}

func (client *ClientHandle) GetId() string {
	var socket_propertiy string
	if client.socketBaseApi == nil {
		socket_propertiy = "socket未初始化"
	} else {
		socket_propertiy = client.socketBaseApi.GetId()
	}
	return fmt.Sprintf("基类%s+注册身份[%s]", socket_propertiy, client.visitorName)
}
func (client *ClientHandle) SetVisitor(name string) {
	client.visitorName = name
}
func (client *ClientHandle) GetVisitor() (name string) {
	return client.visitorName
}
func (client *ClientHandle) SetAutoReconnect() {
	client.autoReconnect = true
}

//CmdUnblockRead = 1
//CmdWrite = 2
//CmdRegisterNotify = 3
//CmdNotify = 4
//CmdHeart = 5
//建立连接,成功返回nil,输入服务器地址，例子:"127.0.0.1:7777"
func (client *ClientHandle) StartConnect(addr string, visitor string, server_request SocketOutputFunc) error {
	client.SetVisitor(visitor)
	client.server_request = server_request
	var err error
	//转换为net.TCPAddr格式
	client.serverAddr, err = net.ResolveTCPAddr("tcp4", addr)
	if err != nil {
		return err
	}
	//建立连接
	client.connection, err = net.DialTCP("tcp4", nil, client.serverAddr)
	if err != nil {
		if client.connection != nil {
			client.connection.Close()
		}
		return err
	}
	//开启协程
	client.socketBaseApi = newRealtimeDataCenterSocket(client.connection, Client_flag, client.server_request, client.CloseNotify)
	client.StartProcess()
	//心跳
	client.子协程退出等待.Add(1)
	go func() {
		defer client.子协程退出等待.Done()
		defer holmes.Errorf("%s心跳协程退出", client.GetId())
		for {
			time.Sleep(time.Second * 1)
			client.Request(CmdHeart, []byte{})
			if client.子协程退出标识符 == true {
				return
			}
		}
	}()
	return nil
}
func (client *ClientHandle) CLoseSocket(reason string) {
	defer func() {
		client.connection = nil
	}()
	if client.connection != nil {
		err := client.connection.Close()
		if err != nil {
			holmes.Errorf("%s主动关闭时错误%s", client.GetId(), err.Error())
		} else {
			holmes.Infof("%s主动关闭时成功,原因%s", client.GetId(), reason)
		}
	} else {
		holmes.Infof("%s已经关闭", client.GetId())
	}
}

//非阻塞读
func (client *ClientHandle) UnblockRead(path string) ReturnWithJsonPayload {
	var ret ReturnWithJsonPayload
	var request visitorBase
	request.Visitor = client.GetVisitor()
	request.Path = path
	//socket
	if client.socketBaseApi != nil {
		ret_bytes, err := client.Request(CmdUnblockRead, []byte(request.String()))
		if err != nil {
			ret.Set(false, fmt.Sprintf("请求通信%s:数据%s", err.Error(), string(ret_bytes)))
			return ret
		} else {
			err := json.Unmarshal(ret_bytes, &ret)
			if err != nil {
				ret.Set(false, fmt.Sprintf("json解析%s:数据%s", err.Error(), string(ret_bytes)))
				return ret
			} else {
				//不用
			}
		}
	} else {
		ret.Set(false, "连接处于关闭中")
	}
	return ret
}

//写
func (client *ClientHandle) WriteData(path string, data interface{}) ReturnBase {
	var ret ReturnBase
	var request changeNotify
	request.Visitor = client.GetVisitor()
	request.Path = path
	send_bytes, err := json.Marshal(data)
	if err != nil {
		ret.Set(false, err.Error())
		return ret
	} else {
		request.Payload = component.ReadResult(string(send_bytes))
	}
	//socket
	if client.socketBaseApi != nil {
		ret_bytes, err := client.Request(CmdWrite, []byte(request.String()))
		if err != nil {
			ret.Set(false, fmt.Sprintf("请求通信%s:数据%s", err.Error(), string(ret_bytes)))
			return ret
		} else {
			err := json.Unmarshal(ret_bytes, &ret)
			if err != nil {
				ret.Set(false, fmt.Sprintf("json解析%s:数据%s", err.Error(), string(ret_bytes)))
				return ret
			} else {
				//不用
			}
		}
	} else {
		ret.Set(false, "连接处于关闭中")
	}
	return ret
}
func (client *ClientHandle) RegisterNotify(path string, timeout int) ReturnWithJsonPayload {
	var ret ReturnWithJsonPayload
	var request visitorRequestNotify
	request.Visitor = client.GetVisitor()
	request.Path = path
	request.Timeout = timeout
	//socket
	if client.socketBaseApi != nil {
		ret_bytes, err := client.Request(CmdRegisterNotify, []byte(request.String()))
		if err != nil {
			ret.Set(false, fmt.Sprintf("请求通信%s:数据%s", err.Error(), string(ret_bytes)))
			return ret
		} else {
			err := json.Unmarshal(ret_bytes, &ret)
			if err != nil {
				ret.Set(false, fmt.Sprintf("json解析%s:数据%s", err.Error(), string(ret_bytes)))
				return ret
			} else {
				//不用
			}
		}
	} else {
		ret.Set(false, "连接处于关闭中")
	}
	return ret
}

//实现重连机制
func (client *ClientHandle) CloseNotify(reason string) {
	client.子协程退出标识符 = true
	//等待所有子协程退出
	client.子协程退出等待.Wait()
	client.socketBaseApi = nil //清空

	client.子协程退出标识符 = false
	if client.autoReconnect {
		//由于可能出现断网情况所以需要多次连接，为了避免无限递归采用协程方式重连
		//在连接每建立成功时，不会开启socket基类，这样也没有socket基类发送关闭通知
		go func() {
			for {
				err := client.StartConnect(client.serverAddr.String(), client.GetVisitor(), client.server_request)
				if err != nil {
					holmes.Errorf("%s连接失败,等待下一次重连", client.GetId())
					time.Sleep(10 * time.Second)
				} else {
					holmes.Infof("%s重连成功", client.GetId())
					return
				}
			}

		}()
	}

}
