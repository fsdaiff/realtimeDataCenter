package component

import (
	"errors"
	"fmt"
	"github.com/fsdaiff/logClient"
	"time"
)

const (
	waitInterval             = time.Second
	defaultWaitTime          = 10
	MaxWaitTime              = 180
	kernelMaxResponseTime    = 500 * time.Millisecond //ms
	kernelMaxSendChannelTime = 100 * time.Millisecond //ms
)

func NewVisitorsControl(json interface{}) *VisitorsControl {
	a := new(VisitorsControl)
	a.JsonObject.SetValue(json)

	a.write = make(chan WriteRequest, 0)
	a.read = make(chan *ReadRequest, 100)
	a.register = make(chan RegisterRequest, 0)
	a.KernalPerformance = getKernalPerformance()
	return a
}

//写入：输入路径，内容 返回error
//读取：阻塞式读取，有限阻塞式读取，非阻塞式读取
//访问管理者:管理访问者、处理访问者的请求并返回结果
type VisitorsControl struct {
	//数据
	JsonObject
	//写入者，不用管理
	write    chan WriteRequest
	read     chan *ReadRequest
	register chan RegisterRequest

	//缓冲区记录
	//read_buffer sync.Map

	//阻塞式读取,计算写影响
	//wait ReadWaitArray
	//有限阻塞式读取,计算写影响，计算时间影响
	waitWithTimeout longPollReadControl
	//非阻塞式读取,不用管理

	//性能监听
	KernalPerformance
}

//受影响的阻塞读取访问者，读取数据并发送,如果发生错误，删除该访问者
//加速:减少相同json重复操作
func (control *VisitorsControl) affectedReadReturn(affected map[string]*VisitorRwaitAndBuffer, visitor_name string, timestamp int64) (string, error) {
	if affected == nil {
		//
		return "", errors.New("输入为空")
	}
	var calculate string
	var err string
	var alreadyJson map[string]ReadResult = make(map[string]ReadResult) //用于节省
	var ret ReadResult
	var ok bool
	for key, value := range affected {
		start := time.Now().UnixNano()
		path_string := value.GetPath().Output2String()
		//已经解析过,用原来的。未解析过的读取核心数据
		if ret, ok = alreadyJson[path_string]; !ok {
			ret = control.Read(value)
			alreadyJson[path_string] = ret
		}
		json_over := time.Now().UnixNano()
		if ret.Nil() {
			err += value.String() + ":读结果返回通道为空\n"
		} else {
			insert_err := value.insertUpdate(ret, visitor_name, timestamp)
			if insert_err != nil {
				err += value.String() + ":更新错误-" + insert_err.Error() + ",删除该访问者\n"
				control.waitWithTimeout.Remove(key)
			}
		}
		end := time.Now().UnixNano()
		calculate += fmt.Sprintf("json操作:%d,channel操作:%d\n", json_over-start, end-json_over)
	}
	if len(err) == 0 {
		return calculate, nil
	} else {
		return calculate, errors.New(err)
	}
}
func (control *VisitorsControl) Control() {
	holmes.Infof("访问管理单元启动")
	defer holmes.Infof("访问管理单元退出")
	holmes.Infof("最大等待间隔%d*%d", MaxWaitTime, waitInterval)
	var start, end, cost_time int64
	for {
		select {
		case rec := <-control.write: //写请求
			start = time.Now().UnixNano()
			//数据实现
			a := new(VisitorW)
			a.Set(rec.visitor_name, rec.road)
			a.SaveData(rec.data)
			select {
			case rec.result <- control.Write(a):
				//holmes.Debugf("向%s写入成功", a.GetPath().Output2String())
			case <-time.After(kernelMaxSendChannelTime):
				//超时错误
				//holmes.Errorf("%s写返回channel超时", rec.visitor_name)
				control.WritePerformance.failRecord("channel超时错误", 1, fmt.Sprintf("%s写返回channel超时", rec.visitor_name))
			}
			write_over := time.Now().UnixNano()
			affected := control.waitWithTimeout.Affect(a.GetPath().Output2String())
			affected_ret := time.Now().UnixNano()
			var err error
			var calculate string
			var affected_len int = len(affected)
			if affected_len > 0 {
				calculate, err = control.affectedReadReturn(affected, rec.visitor_name, write_over)
				if err != nil {
					//holmes.Errorln(err)
					control.WriteAffectedPerformance.failRecord("处理错误", affected_len, fmt.Sprintf("错误:%s\n%s", err.Error(), calculate))
				}
			}
			end = time.Now().UnixNano()
			control.WriteAffectedPerformance.add(affected_len, end-write_over)
			cost_time = (end - start)
			control.WritePerformance.add(1, cost_time)
			if cost_time > int64(kernelMaxResponseTime) {
				var summary string
				summary += fmt.Sprintf("核心节点写请求处理时间过长,请求参数%v\n", rec)
				summary += fmt.Sprintf("总时间:%d,写时间:%d,影响处理:%d,影响返回:%d\n", end-start, write_over-start, end-write_over, end-affected_ret)
				summary += fmt.Sprintf("具体统计时间\n%s", calculate)
				//holmes.Errorf(summary)
				control.WritePerformance.failRecord("处理时间过长", 1, summary)
			}
		case rec := <-control.read: //非阻塞读处理
			start = time.Now().UnixNano()
			read := CreateRead(*rec)
			select {
			case read.GetReturn() <- control.Read(read):
			case <-time.After(kernelMaxSendChannelTime):
				//超时错误
				summary := fmt.Sprintf("%s不阻塞读返回channel超时", rec.visitor_name)
				//holmes.Errorf(summary)
				control.UnblockReadPerformance.failRecord("channel超时错误", 1, summary)
			}
			end = time.Now().UnixNano()
			cost_time = (end - start)
			control.UnblockReadPerformance.add(1, cost_time)
			if cost_time > int64(kernelMaxResponseTime) {
				summary := fmt.Sprintf("核心节点读请求处理时间过长,请求参数%v", rec)
				//holmes.Errorf(summary)
				control.UnblockReadPerformance.failRecord("处理时间过长", 1, summary)
			}

		case rec := <-control.register: //资源监听注册,返回空为失败,表示不存在
			start = time.Now().UnixNano()
			var result ReadResult
			var road Path
			road.InputFromString(rec.road)
			//查找路径是否存在
			find_result := control.Find(road)
			if find_result == nil { //不存在
				result = NilKey
				control.RegiterPerformance.failRecord("请求资源不存在", 1, fmt.Sprintf("不存在路径%s", rec.road))
			} else {
				//申请缓冲区资源
				visitor_buf := control.waitWithTimeout.applyFor(CreateVisitorRwaitFromRegisterRequest(rec))
				if visitor_buf != nil {
					result = ReadResult("成功")
				} else {
					control.RegiterPerformance.failRecord("无效资源", 1, fmt.Sprintf("无效缓冲区管道%s", rec))
					result = NilKey
				}
			}
			select {
			case rec.result <- result:
			case <-time.After(kernelMaxSendChannelTime):
				//holmes.Errorf("注册请求%s结果通过chan返回缓冲区超时", rec)
				control.RegiterPerformance.failRecord("channel超时错误", 1, fmt.Sprintf("注册请求%s结果通过chan返回缓冲区超时", rec))
			}
			end = time.Now().UnixNano()
			cost_time = (end - start)
			control.RegiterPerformance.add(1, cost_time)
			if cost_time > int64(kernelMaxResponseTime) {
				summary := fmt.Sprintf("核心节点注册请求处理时间过长,请求参数%v", rec)
				//holmes.Errorf(summary)
				control.RegiterPerformance.failRecord("处理时间过长", 1, summary)
			}
		case <-time.After(time.Millisecond * 5000): //轮询处理
		}
	}

}

//read
func (control *VisitorsControl) ReadWithTime(visitor_name string, road string, timeout int) ReadBufferResult {
	a := newReadRequest(visitor_name, road, timeout)
	if timeout <= 0 {
		holmes.Errorf("ReadWithTime，timeout错误%d", timeout)
		a.timeout = defaultWaitTime
	}
	//检查资源是否存在
	var path1 Path
	path1.InputFromString(road)
	ptr := control.Find(path1)
	if ptr == nil {
		holmes.Errorf("不存在资源路径", road)
		return DefaultReadBufferResult()
	} else {
		return control.ReadBuffer(visitor_name, road, timeout)
	}
	/*select {
	case control.read <- a:
		select {
		case buffer_ptr := <-a.buffer_ptr:
			if buffer_ptr != nil {
				result := buffer_ptr.readOne(timeout)
				//读缓冲区成功
				if !result.Nil() {
					return result
				} else {
					//超时
					return control.ReadQuickly(visitor_name, road)
				}
			} else {
				holmes.Errorf("%s读请求返回缓冲区指针为空", visitor_name)
				return NilKey
			}
		case <-time.After(kernelMaxResponseTime):
			holmes.Errorf("%s等待读请求返回缓冲区指针超时", visitor_name)
			return NilKey
		}
	case <-time.After(kernelMaxResponseTime * 2):
		holmes.Errorf("%s发送读请求返回超时", visitor_name)
		return NilKey
	}
	return NilKey*/
}
func (control *VisitorsControl) RegisterNotify(visitor_name string, road string, timeout int) ReadResult {
	a := newRegisterRequest(visitor_name, road, timeout)
	if timeout <= 0 {
		holmes.Errorf("RegisterNotify，timeout错误%d", timeout)
		a.timeout = defaultWaitTime
	}
	select {
	case control.register <- *a:
		select {
		case result := <-a.result:
			return result
		case <-time.After(kernelMaxResponseTime):
			holmes.Errorf("%s等待读请求返回缓冲区指针超时", visitor_name)
			return NilKey
		}
	case <-time.After(kernelMaxResponseTime * 2):
		holmes.Errorf("%s发送注册请求返回超时", a)
		return NilKey
	}
	return NilKey
}

//在读ReadBuffer之前，首先要确定road是否存在
func (control *VisitorsControl) ReadBuffer(visitor_name string, road string, timeout int) ReadBufferResult {
	var start int64 = time.Now().UnixNano()
	a := newRegisterRequest(visitor_name, road, timeout)
	defer func() {
		end := time.Now().UnixNano()
		control.LongPollReadPerformance.add(1, end-start)
	}()

	if timeout <= 0 {
		holmes.Errorf("RegisterNotify，timeout错误%d", timeout)
		a.timeout = defaultWaitTime
	}
	//申请缓冲区资源
	buffer_ptr := control.waitWithTimeout.applyFor(CreateVisitorRwaitFromRegisterRequest(*a))
	if buffer_ptr != nil {
		result := buffer_ptr.readOne(timeout)
		//读缓冲区成功
		if !result.Nil() {
			return result
		} else {
			//超时
			var defaultResult ReadBufferResult
			defaultResult.Set(NilVisitor, control.ReadQuickly(visitor_name, road), time.Now().UnixNano())
			return defaultResult
		}
	} else {
		holmes.Errorf("%s读请求返回缓冲区指针为空", visitor_name)
		return DefaultReadBufferResult()
	}

}

//read
func (control *VisitorsControl) ReadQuickly(visitor_name string, road string) ReadResult {
	a := newReadRequest(visitor_name, road, -1)
	start := time.Now().UnixNano()
	var request_over, ret_over int64
	defer func() {
		end := time.Now().UnixNano()
		holmes.Debugf("一次读取总时间%d,申请时间:%d,核心处理时间:%d", end-start, request_over-start, ret_over-request_over)
	}()
	select {
	case control.read <- a:
		request_over = time.Now().UnixNano()
		select {
		case result := <-a.result:
			ret_over = time.Now().UnixNano()
			return result
		case <-time.After(time.Millisecond * 100):
			holmes.Errorf("%s等待读请求返回超时", visitor_name)
			return NilKey
		}
	case <-time.After(time.Millisecond * 1000):
		holmes.Errorf("%s发送读请求返回超时", visitor_name)
		return NilKey
	}
	return NilKey
}

//write
func (control *VisitorsControl) WriteQuickly(visitor_name string, road string, data interface{}) error {
	a := GetNewWriteRequest(visitor_name, road, data)
	from := time.Now().UnixNano()
	select {
	case control.write <- a:
		request_over := time.Now().UnixNano()
		select {
		case result := <-a.result:
			to := time.Now().UnixNano()
			holmes.Debugf("起始时间:%d,结束时间：%d,总时间:%d,请求时间:%d,发送时间:%d-写", from, to, to-from, request_over-from, to-request_over)
			return result
		case <-time.After(time.Millisecond * 100):
			return errors.New("写操作等待返回超时")
		}
	case <-time.After(time.Millisecond * 1000):
		return errors.New("内部管道传送失败")
	}
}
func (control VisitorsControl) PerformanceShow() string {
	return control.KernalPerformance.String()
}

//
func GetNewWriteRequest(visitor_name string, road string, data interface{}) WriteRequest {
	var a WriteRequest

	a.visitor_name = visitor_name
	a.road = road
	a.data = data
	a.result = make(chan error, 0)

	return a
}

type basicRequest struct {
	visitor_name string
	road         string
}
type WriteRequest struct {
	basicRequest
	data   interface{}
	result chan error
}

//用于注册监听
func newRegisterRequest(visitor_name string, road string, timeout int) *RegisterRequest {
	a := new(RegisterRequest)

	a.visitor_name = visitor_name
	a.road = road
	a.timeout = timeout
	a.result = make(chan ReadResult)
	return a
}

type RegisterRequest struct {
	basicRequest
	timeout int
	result  chan ReadResult //用于立即返回
}

func (register RegisterRequest) String() string {
	return fmt.Sprintf("[visitor:%s,path:%s,timeout:%d]", register.visitor_name, register.road, register.timeout)
}
func newReadRequest(visitor_name string, road string, timeout int) *ReadRequest {
	a := new(ReadRequest)

	a.visitor_name = visitor_name
	a.road = road
	a.timeout = timeout
	a.result = make(chan ReadResult)
	a.buffer_ptr = make(chan buffer4read)
	return a
}

type ReadRequest struct {
	basicRequest
	timeout    int
	result     chan ReadResult  //用于立即返回
	buffer_ptr chan buffer4read //用于阻塞式返回，获取缓冲通道
}

type ControlWrite interface {
	WriteQuickly(visitor_name string, road string, data interface{}) error
}
type ControlRead interface {
	//-1：非阻塞式读取
	//0:阻塞式读取
	ReadQuickly(visitor_name string, road string) ReadResult
	//注册
	RegisterNotify(visitor_name string, road string, timeout int) ReadResult
	//用于socket的读,使用时需谨慎，因为其没有进行资源检查
	ReadBuffer(visitor_name string, road string, timeout int) ReadBufferResult
	//>0:有限阻塞式读取,单位ms
	ReadWithTime(visitor_name string, road string, timeout int) ReadBufferResult
	//性能统计
	PerformanceShow() string
}
