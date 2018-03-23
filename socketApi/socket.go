package socketApi

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/fsdaiff/logClient"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const FrameStart = 0x68
const FrameEnd = 0x16
const Version = 0x10
const Server_flag = true
const Client_flag = false
const ServerRequest = 0x10
const ServerAck = 0x11
const ClientRequest = 0x00
const ClientAck = 0x01
const CmdUnblockRead = 1
const CmdWrite = 2
const CmdRegisterNotify = 3
const CmdNotify = 4
const CmdHeart = 5

//socket回调函数定义
type SocketOutputFunc func(cmd byte, payload []byte) []byte
type CloseNotify func(reason string)

//产生socket处理基类
func newRealtimeDataCenterSocket(connection *net.TCPConn, serverOrClient bool, notify SocketOutputFunc, close_notify CloseNotify) *realtimeDataCenterSocket {
	a := new(realtimeDataCenterSocket)
	//a.SetId(connection.RemoteAddr().String())
	//临界部分
	a.等待主线程启动完成.Add(1)
	a.等待上层发送启动处理.Add(1)
	go a.SocketPoll(connection, serverOrClient, notify, close_notify)
	a.waitSocketPollStart()
	return a
}

//并行转串行
//socket 处理基础类
type realtimeDataCenterSocket struct {
	id    string //识别码
	order uint64
	//serverOrClient bool
	request chan requestFromSocket
	//connection     *net.TCPConn //连接
	//output         SocketOutputFunc
	//close_notify   CloseNotify
	ret_queue  sync.Map //对方应答队列
	等待主线程启动完成  sync.WaitGroup
	等待上层发送启动处理 sync.WaitGroup
}

//每个socket 处理基础类的id
func (onesocket *realtimeDataCenterSocket) GetId() string {
	return onesocket.id
}
func (onesocket *realtimeDataCenterSocket) SetId(id string) {
	onesocket.id = id
}
func (onesocket *realtimeDataCenterSocket) waitSocketPollStart() {
	onesocket.等待主线程启动完成.Wait()
}

//提供给上层的发起请求函数
func (onesocket *realtimeDataCenterSocket) Request(cmd byte, payload []byte) (ret []byte, err error) {
	var req requestFromSocket
	req.order = atomic.AddUint64(&onesocket.order, 1)
	req.payload = payload
	req.cmd = cmd
	req.ret = make(chan retFromSocket, 0)
	select {
	case onesocket.request <- req:
		select {
		case ret := <-req.ret:
			return ret.data, ret.err
		case <-time.After(time.Millisecond * 500):
			return nil, errors.New(fmt.Sprintf("[%s]-[%d]发起请求成功,等待相应时超时\n%s", onesocket.id, req.order, string(req.payload)))
		}
	case <-time.After(time.Millisecond * 500):
		return nil, errors.New(fmt.Sprintf("[%s]-[%d]发起请求超时\n%s", onesocket.id, req.order, string(req.payload)))
	}
	return nil, errors.New("未知")
}
func (onesocket *realtimeDataCenterSocket) StartProcess() {
	onesocket.等待上层发送启动处理.Done()
}

//socket编码解码协程，负责上层数据和socket数据的转换
func (onesocket *realtimeDataCenterSocket) SocketPoll(connection *net.TCPConn, serverOrClient bool, notify SocketOutputFunc, close_notify CloseNotify) {
	//初始化
	onesocket.request = make(chan requestFromSocket, 0)
	var read_err_exit chan bool = make(chan bool, 0)
	var return2socket chan ret2Socket = make(chan ret2Socket, 0) //对方请求后，向socket写的应答数据
	var heartBeat int64 = time.Now().Unix()
	var reason string
	onesocket.SetId(fmt.Sprintf("[本程序:%s<-->访问者:%s]", connection.LocalAddr(), connection.RemoteAddr()))
	defer func() {
		if close_notify != nil {
			close_notify(fmt.Sprintf("[%s]关闭%s", onesocket.GetId(), reason))
		}
	}()
	onesocket.等待主线程启动完成.Done()
	onesocket.等待上层发送启动处理.Wait()
	holmes.Debugf("%s主线程启动完成", onesocket.GetId())
	go func() {
		defer func() {
			if connection != nil {
				connection.Close()
				select {
				case read_err_exit <- true:
				case <-time.After(time.Millisecond * 500):
					holmes.Errorf("[%s]通知read关闭超时", onesocket.GetId())
				}
			}
		}()
		for {
			var typeData []byte = make([]byte, 22)
			//读帧头
			n, err := io.ReadFull(connection, typeData)
			if err != nil {
				holmes.Errorf("[%s]read错误%s,数据%x", onesocket.GetId(), err, typeData[:n])
				return
			}
			if typeData[0] != FrameStart {
				holmes.Errorf("[%s]read错误,数据头不为%x,数据为%x", onesocket.GetId(), FrameStart, typeData)
				return
			}
			if typeData[1] != Version {
				holmes.Errorf("[%s]read错误,数据头不为%x,数据为%x", onesocket.GetId(), FrameStart, typeData)
				return
			}
			typeBuf := bytes.NewReader(typeData[2:])
			var timeStamp int64
			if err := binary.Read(typeBuf, binary.LittleEndian, &timeStamp); err != nil {
				holmes.Errorf("[%s]read时间戳错误%s,数据为%x", onesocket.GetId(), err.Error(), typeData)
				return
			}
			var direction uint8
			if err := binary.Read(typeBuf, binary.LittleEndian, &direction); err != nil {
				holmes.Errorf("[%s]read方向错误%s,数据为%x", onesocket.GetId(), err.Error(), typeData)
				return
			}
			var order uint64
			if err := binary.Read(typeBuf, binary.LittleEndian, &order); err != nil {
				holmes.Errorf("[%s]read序号错误%s,数据为%x", onesocket.GetId(), err.Error(), typeData)
				return
			}
			var cmd byte
			if err := binary.Read(typeBuf, binary.LittleEndian, &cmd); err != nil {
				holmes.Errorf("[%s]read命令错误%s,数据为%x", onesocket.GetId(), err.Error(), typeData)
				return
			}
			var length uint16
			if err := binary.Read(typeBuf, binary.LittleEndian, &length); err != nil {
				holmes.Errorf("[%s]read数据长度错误%s,数据为%x", onesocket.GetId(), err.Error(), typeData)
				return
			}
			var realData []byte = make([]byte, length+1)
			n, err = io.ReadFull(connection, realData)
			if err != nil {
				holmes.Errorf("[%s]read具体数据和帧尾错误%s,数据%x", onesocket.GetId(), err, realData[:n])
				return
			}
			//帧尾
			if realData[length] != FrameEnd {
				holmes.Errorf("[%s]read错误,数据头不为%x,数据为%x", onesocket.GetId(), FrameStart, typeData)
				return
			}
			var payload []byte = realData[:length]
			//请求或应答

			switch direction {
			case ServerRequest:
				fallthrough
			case ClientRequest: //向上层请求
				if serverOrClient == Server_flag && direction == ServerRequest {
					holmes.Errorf("[%s]帧方向错误%x,数据为%x%x", onesocket.GetId(), direction, typeData, realData)
				} else {
					var retStruct ret2Socket
					retStruct.data = make([]byte, 0)
					retStruct.order = order
					retStruct.cmd = cmd
					if retStruct.cmd == CmdHeart { //心跳处理
						heartBeat = time.Now().Unix()
					} else if notify != nil {
						ret := notify(cmd, payload)
						retStruct.data = ret
					} else {
						holmes.Errorf("[%s]请求回调未注册", onesocket.GetId())
						continue
					}
					select {
					case return2socket <- retStruct:
					case <-time.After(time.Millisecond * 500):
						holmes.Errorf("[%s]应答通道阻塞超时", onesocket.GetId())
					}
				}

			case ServerAck:
				fallthrough
			case ClientAck: //对方应答
				if serverOrClient == Server_flag && direction == ServerAck {
					holmes.Errorf("[%s]帧方向错误%x,数据为%x%x", onesocket.GetId(), direction, typeData, realData)
				} else {
					//返回给上层
					if value, ok := onesocket.ret_queue.Load(order); ok {
						if value.(requestFromSocket).ret != nil {
							var r1 retFromSocket
							r1.data = payload
							select {
							case value.(requestFromSocket).ret <- r1:
							case <-time.After(time.Millisecond * 500):
								holmes.Errorf("[%s]%d号请求返回到上层时超时", onesocket.GetId(), order)
							}
							//删除
							onesocket.ret_queue.Delete(order)
						} else {
							holmes.Errorf("[%s]%d号请求返回通道为空", onesocket.GetId(), order)
						}

					} else {
						holmes.Errorf("[%s]不存在%d", onesocket.GetId(), order)
					}
				}
			default:
				holmes.Errorf("[%s]帧方向错误%x,数据为%x%x", onesocket.GetId(), direction, typeData, realData)
			}
		}
	}()
	for {
		var order uint64
		var ret []byte
		var direction uint8
		var cmd byte
		select {
		case <-read_err_exit:
			return
		case heighLevelReturn := <-return2socket: //上层应答
			ret = heighLevelReturn.data
			order = heighLevelReturn.order
			cmd = heighLevelReturn.cmd
			if serverOrClient == Server_flag {
				direction = ServerAck
			} else {
				direction = ClientAck
			}

		case request := <-onesocket.request: //上层请求
			ret = request.payload
			order = request.order
			cmd = request.cmd
			if serverOrClient == Server_flag {
				direction = ServerRequest
			} else {
				direction = ClientRequest
			}
			//储存上层请求，为应答提供指示
			onesocket.ret_queue.LoadOrStore(order, request)
		case <-time.After(time.Second * 2):
			now := time.Now().Unix()
			if now-heartBeat >= 3 {
				connection.Close()
				holmes.Errorf("[%s]心跳超时%d", onesocket.GetId(), now-heartBeat)
				return
			}
			continue
		}
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.LittleEndian, byte(FrameStart))
		binary.Write(buf, binary.LittleEndian, byte(Version))
		binary.Write(buf, binary.LittleEndian, int64(time.Now().UnixNano()/1000))
		binary.Write(buf, binary.LittleEndian, byte(direction))
		binary.Write(buf, binary.LittleEndian, order)
		binary.Write(buf, binary.LittleEndian, cmd)
		binary.Write(buf, binary.LittleEndian, uint16(len(ret)))
		binary.Write(buf, binary.LittleEndian, ret)
		binary.Write(buf, binary.LittleEndian, byte(FrameEnd))
		packet := buf.Bytes()
		_, err := connection.Write(packet)
		if err != nil {
			holmes.Errorf("[%s]发送错误%s\n%x", onesocket.GetId(), err.Error(), packet)
			connection.Close()
			return
		}

	}
}

type socketBaseApi interface {
	StartProcess() //防止上层未初始化完成时，发生回调
	SocketPoll(connection *net.TCPConn, serverOrClient bool, notify SocketOutputFunc, close_notify CloseNotify)
	Request(cmd byte, payload []byte) (ret []byte, err error)
	GetId() string
	SetId(id string)
}
