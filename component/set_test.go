package component

import (
	"github.com/json-iterator/go"
	"strconv"
	"testing"
	"time"
)

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
var input interface{}
var road string
var child interface{}

func Test_createStdBtree(t *testing.T) {
	var a JsonObject
	var path Path
	a.SetValue(CreateStdBtree(2, 0, 2))
	road = CreateBtreeRoad2leaf(2)
	path.InputFromString(road)
	result := a.Find(path)
	if result == nil {
		t.Log(a)
		t.Error("未发现" + path.Output2String())
	} else {
		str, _ := jsoniter.MarshalToString(a.value)
		t.Logf("CreateStdBtree正确创建:%s", str)
		child = result
	}

	var i, j int
	bianli7(a.value.(map[string]interface{}), &i)
	t.Logf("所有节点数:%d", i)
	bianli8(a.value.(map[string]interface{}), &j)
	t.Logf("所有叶节点数:%d", j)

	t.Logf("赋值到全局变量中")
}

///////////////////////////////////////////////

func Test_write_read(t *testing.T) {
	jsoniter.Unmarshal([]byte(test_json), &true_input)
	jsoniter.Unmarshal([]byte(test_json2), &true_input2)
	control := NewVisitorsControl(true_input)
	var b ControlWrite
	b = control
	go func() {
		control.Control()
	}()
	time.Sleep(100 * time.Millisecond)
	err := b.WriteQuickly("dgl", "data/config_list/reg_conf/s", true_input2)
	t.Logf("写%s", test_json2)
	if err != nil {
		t.Error(err)
	}
	var c ControlRead
	c = control

	d := c.ReadQuickly("read", "data/config_list/reg_conf/s")
	e, _ := jsoniter.MarshalToString(d)
	t.Logf("读结果:%s", e)
}
func Benchmark_WriteRead(b *testing.B) {
	var a JsonObject
	var path Path

	input = CreateStdBtree(4, 0, 3)
	a.SetValue(input)
	road = CreateBtreeRoad2leaf(3)
	path.InputFromString(road)
	child = a.Find(path)

	control := NewVisitorsControl(input)
	go func() {
		control.Control()
	}()
	//str, _ := jsoniter.MarshalToString(control.value)
	//b.Logf("表:%s\n路径:%s", str, road)
	var w ControlWrite = control
	var r ControlRead = control
	child = r.ReadQuickly("read", road)
	b.StopTimer() //调用该函数停止压力测试的时间计数

	//做一些初始化的工作,例如读取文件数据,数据库连接之类的,
	//这样这些时间不影响我们测试函数本身的性能

	b.StartTimer() //重新开始时间
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w.WriteQuickly("dgl", road, child)
		r.ReadQuickly("read", road)
	}
}
func Benchmark_Write(b *testing.B) {

	control := NewVisitorsControl(input)
	go func() {
		control.Control()
	}()
	var w ControlWrite = control
	//var r ControlRead = control

	b.StopTimer() //调用该函数停止压力测试的时间计数

	//做一些初始化的工作,例如读取文件数据,数据库连接之类的,
	//这样这些时间不影响我们测试函数本身的性能

	b.StartTimer() //重新开始时间
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w.WriteQuickly("dgl", road, child)
		//r.ReadQuickly("read", "data/config_list/reg_conf/s")
	}
}
func Benchmark_Read(b *testing.B) {

	control := NewVisitorsControl(input)
	go func() {
		control.Control()
	}()
	var w ControlWrite = control
	var r ControlRead = control
	w.WriteQuickly("dgl", road, child)
	b.StopTimer() //调用该函数停止压力测试的时间计数

	//做一些初始化的工作,例如读取文件数据,数据库连接之类的,
	//这样这些时间不影响我们测试函数本身的性能

	b.StartTimer() //重新开始时间
	for i := 0; i < b.N; i++ {

		r.ReadQuickly("read", road)
	}
	b.ReportAllocs()
}
func Benchmark_JsonitorDecode(b *testing.B) {
	str, _ := jsoniter.Marshal(input)

	b.StopTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		jsoniter.Unmarshal(str, &input)
	}
	b.ReportAllocs()
}
func Benchmark_JsonitorEncode(b *testing.B) {
	b.StopTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		jsoniter.Marshal(input)
	}
	b.ReportAllocs()
}
