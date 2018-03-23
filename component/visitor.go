package component

//多线程json树访问
//1、树某个子树的多线程写请求处理
//2、树某个子树的多线程读请求处理
//3、计算写的影响，解锁读阻塞
import (
	"errors"
	"fmt"
	"github.com/fsdaiff/logClient"
	"sync/atomic"
	"time"
)

const NilKey = ""
const NilVisitor = ""
const Niltimestamp = 0

type ReadResult string

//判断是否为空
func (result ReadResult) Nil() bool {
	if result == NilKey {
		return true
	} else {
		return false
	}
}
func DefaultReadBufferResult() ReadBufferResult {
	var a ReadBufferResult
	a.Set(NilVisitor, NilKey, Niltimestamp)
	return a
}

//读缓冲区的数据
type ReadBufferResult struct {
	result      ReadResult
	w_timestamp int64 //ns
	writer      string
}

func (ret ReadBufferResult) Nil() bool {
	return ret.result.Nil()
}
func (ret *ReadBufferResult) Set(writer string, result ReadResult, timestamp int64) {
	ret.result = result
	ret.writer = writer
	ret.w_timestamp = timestamp
}
func (ret ReadBufferResult) Get() (writer string, result ReadResult, timestamp int64) {
	return ret.writer, ret.result, ret.w_timestamp
}

//访问者类型定义
const (
	VisitorWriteNow             = iota
	VisitorReadWait             = iota
	VisitorReadWaitUntilTimeout = iota
	VisitorReadReturnNow        = iota
)

//请求基本信息
type Visitor interface {
	Name() string                 //请求的名称,用于身份识别
	String() string               //用于打印
	Set(name string, path string) //初始化,设置数据
	GetPath() Path                //要访问的节点在树中的路径
}

//请求、写入者
type VisitorWrite interface {
	Visitor                    //请求基础类
	SaveData(data interface{}) //写入数据
	GetData() interface{}      //获取写入的数据
}

//请求:读取者
type VisitorRead interface {
	Visitor                         //请求基础类
	SaveReturn(ret chan ReadResult) //请求者设置返回channel
	GetReturn() chan ReadResult     //json管理者，获取要返回请求者的channel
}

//////////////////////////////////////////////////////////
type VisitorBasicType struct {
	path Path
	name string
}

//初始化
func (read *VisitorBasicType) Set(name string, path string) {
	read.name = name
	read.path.InputFromString(path)
}

//打印时
func (read VisitorBasicType) String() string {
	return "[访问者:" + read.name + "+访问路径:" + read.path.Output2String() + "]"
}
func (read VisitorBasicType) Name() string {
	return read.name
}
func (read VisitorBasicType) GetPath() Path {
	return read.path
}

//////////////////////////////////////////////////////////
type VisitorW struct {
	VisitorBasicType
	data interface{}
}

func (write *VisitorW) SaveData(data interface{}) {
	write.data = data
}
func (write *VisitorW) GetData() interface{} {
	return write.data
}

//////////////////////////////////////////////////////////
type VisitorRwait struct {
	VisitorBasicType
	ret chan ReadResult
}

func (read *VisitorRwait) SaveReturn(ret chan ReadResult) {
	read.ret = ret
}
func (read *VisitorRwait) GetReturn() chan ReadResult {
	return read.ret
}

//带写入缓冲区的读访问者
type VisitorRwaitAndBuffer struct {
	readNum  int64 //读缓冲区次数
	writeNum int64 //写缓冲区次数
	VisitorRwait
	buffer chan ReadBufferResult
}

//插入新数据
func (visitor *VisitorRwaitAndBuffer) insertUpdate(data ReadResult, visitor_name string, timestamp int64) error {
	//计数加一
	atomic.AddInt64(&visitor.writeNum, 1)
	var a ReadBufferResult
	a.Set(visitor_name, data, timestamp)
	select {
	case visitor.buffer <- a:
		return nil
	case <-time.After(10 * time.Millisecond):
		if len(visitor.buffer) == cap(visitor.buffer) {
			return errors.New("读访问者" + visitor.String() + "缓冲区满" + fmt.Sprintf("%d", len(visitor.buffer)))
		} else {
			return errors.New("读访问者" + visitor.String() + "写入缓冲区时未知错误")
		}

	}
}
func (visitor *VisitorRwaitAndBuffer) readOne(timeout int) ReadBufferResult {
	//计数加一
	atomic.AddInt64(&visitor.readNum, 1)
	select {
	case result := <-visitor.buffer:
		return result
	case <-time.After(waitInterval * time.Duration(timeout)):
		read_num, write_num := visitor.count()
		holmes.Errorf("%s读缓冲区超时，没有写入者,读次数:%d,写次数:%d", visitor.String(), read_num, write_num)
		return DefaultReadBufferResult()
	}
}
func (visitor *VisitorRwaitAndBuffer) count() (read int64, write int64) {
	return atomic.LoadInt64(&visitor.readNum), atomic.LoadInt64(&visitor.writeNum)
}

type buffer4read interface {
	String() string                   //身份特征
	readOne(timeout int) ReadResult   //读缓冲区，超时返回空
	count() (read int64, write int64) //读写计数
}

//////////////////////////////////////////////////////////
//ms
func CreateRead(input ReadRequest) *VisitorRwait {
	visitor := new(VisitorRwait)
	visitor.Set(input.visitor_name, input.road)
	visitor.SaveReturn(input.result)
	return visitor
}
func CreateVisitorRwait(input ReadRequest) VisitorRwait {
	var visitor VisitorRwait
	visitor.Set(input.visitor_name, input.road)
	visitor.SaveReturn(input.result)
	return visitor
}
func CreateVisitorRwaitFromRegisterRequest(input RegisterRequest) VisitorRwait {
	var visitor VisitorRwait
	visitor.Set(input.visitor_name, input.road)
	visitor.SaveReturn(input.result)
	return visitor
}

func newVisitorRwaitAndBuffer(visitor VisitorRwait) *VisitorRwaitAndBuffer {
	a := new(VisitorRwaitAndBuffer)
	a.VisitorRwait = visitor
	a.buffer = make(chan ReadBufferResult, 500)
	return a
}
