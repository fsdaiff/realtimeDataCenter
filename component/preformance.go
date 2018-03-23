package component

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fsdaiff/logClient"
	"sync/atomic"
	"time"
)

type cycleRecord struct {
	records   [100]string
	cycle_num int
}

func (cycle *cycleRecord) record(reason string) {
	cycle.records[cycle.cycle_num] = reason
	cycle.cycle_num += 1
	if cycle.cycle_num >= 100 {
		cycle.cycle_num = 0
	}
}
func (cycleRecord *cycleRecord) getRecord() string {
	var ret string
	for i := 0; i < 100; i++ {
		if cycleRecord.records[i] == "" {
			break
		}
		ret += fmt.Sprintf("-----------------\n%s\n", cycleRecord.records[i])
	}
	return ret
}

//性能记录
//统计数据
//运行时长

//次数
//告警次数
//map[失败类型]失败次数
//最近100条失败记录
//最大值，最小值,平均时间
//最近一分钟的写次数
//一天内操作次数,按小时

//输入
//Add(n,耗时)
//Fail(类型，次数,原因)
//Warn(类型，次数,原因)
//某操作的性能记录
type singleOperation interface {
	add(number_operation int, cost_time int64)
	failRecord(failtype string, number_fail int, reason string)
	warnRecord(warntype string, number_warn int, reason string)
	recreate() //计算显示
}

func new_singleNodePerformance() *singleNodePerformance {
	a := new(singleNodePerformance)
	a.WarnNum = make(map[string]int64, 20)
	a.FailNum = make(map[string]int64, 20)
	return a
}

type singleNodePerformance struct {
	OperationNum int64 `json:"操作次数"`

	WarnNum    map[string]int64 `json:"警告次数"`
	WarnRecord string           `json:"警告记录"`
	warncycle  cycleRecord

	FailNum    map[string]int64 `json:"失败次数"`
	FailRecord string           `json:"失败记录"`
	failcycle  cycleRecord

	MinOpt     int64     `json:"最短操作时间ns"` //最小值,平均时间,最大值
	AverageOpt int64     `json:"平均操作时间ns"`
	MaxOpt     int64     `json:"最长操作时间ns"`
	OnedayOpt  [24]int64 `json:"一天内操作次数"`
}

func (node *singleNodePerformance) recreate() {
	node.FailRecord = node.failcycle.getRecord()
	node.WarnRecord = node.warncycle.getRecord()
}
func (node *singleNodePerformance) add(number_operation int, cost_time int64) {
	if number_operation <= 0 {
		return
	}
	//总记录
	opt_num := atomic.AddInt64(&node.OperationNum, int64(number_operation))
	average_cost_time := cost_time / int64(number_operation)
	//平均
	if node.MinOpt > average_cost_time || node.MinOpt == 0 {
		node.MinOpt = average_cost_time
	}
	if node.MaxOpt < average_cost_time {
		node.MaxOpt = average_cost_time
	}
	node.AverageOpt = (node.AverageOpt*(opt_num-1) + cost_time) / opt_num

	//一天内操作次数
	node.OnedayOpt[time.Now().Hour()] += int64(number_operation)
}
func (node *singleNodePerformance) failRecord(failtype string, number_fail int, reason string) {
	defer holmes.Errorf("%s--%s", failtype, reason)
	if node.FailNum == nil {
		node.FailNum = make(map[string]int64, 20)
	}
	node.FailNum[failtype] += int64(number_fail)
	node.failcycle.record(reason)
}
func (node *singleNodePerformance) warnRecord(warntype string, number_warn int, reason string) {
	defer holmes.Warnf("%s--%s", warntype, reason)
	if node.WarnNum == nil {
		node.WarnNum = make(map[string]int64, 20)
	}
	node.WarnNum[warntype] += int64(number_warn)
	node.warncycle.record(reason)
}
func getKernalPerformance() KernalPerformance {
	var a KernalPerformance
	a.startTime = time.Now().Unix()
	a.UnblockReadPerformance = new_singleNodePerformance()
	a.WritePerformance = new_singleNodePerformance()
	a.LongPollReadPerformance = new_singleNodePerformance()
	a.WriteAffectedPerformance = new_singleNodePerformance()
	a.RegiterPerformance = new_singleNodePerformance()
	return a
}

type KernalPerformance struct {
	startTime int64
	Runtimme  string `json:"运行时长"`

	WritePerformance         singleOperation `json:"写"`
	LongPollReadPerformance  singleOperation `json:"长轮询读"`
	WriteAffectedPerformance singleOperation `json:"写读缓冲区"`
	RegiterPerformance       singleOperation `json:"socket注册监听"`
	UnblockReadPerformance   singleOperation `json:"非阻塞读"`
}

func (performance KernalPerformance) String() string {
	performance.Runtimme = fmt.Sprintf("%ds", (time.Now().Unix() - performance.startTime))
	performance.UnblockReadPerformance.recreate()
	performance.WritePerformance.recreate()
	performance.LongPollReadPerformance.recreate()
	performance.WriteAffectedPerformance.recreate()
	performance.RegiterPerformance.recreate()
	return indent2string(performance)
}

func indent2string(a interface{}) string {
	datas, _ := json.Marshal(a)
	var out bytes.Buffer
	json.Indent(&out, datas, "", "\t")
	return out.String()
}
