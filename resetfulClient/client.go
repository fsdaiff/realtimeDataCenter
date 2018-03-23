package dataCenterClient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"logClient"
	"net/http"
)

//服务器地址
var serverAddr string = "http://wide.fgy:8083/v1.0/realtime"

func SetServerAddr(addr string) {
	serverAddr = addr
}

//访问者名称
var visitorName string = "dgl"

func SetvisitorName(name string) {
	visitorName = name
}

type ReturnBase struct {
	Status string `json:"status"` //ok
	Reason string `json:"reason"`
}

//成功:true 失败:false
func (ret *ReturnBase) Set(success bool, fail_reason string) {
	if success {
		ret.Status = "ok"
	} else {
		ret.Status = "fail"
		ret.Reason = fail_reason
	}
}

//根据返回判断数据正确
func (ret ReturnBase) Success() bool {
	if ret.Status == "ok" {
		return true
	} else {
		return false
	}
}

type ReturnWithPayload struct {
	ReturnBase
	Payload interface{} `json:"data"`
}

//阻塞式读取
func HttpReadWithTimeout(path string, timeout int, payloadType interface{}) (ret ReturnWithPayload) {
	resp, err := http.Get(fmt.Sprintf("%s/%s?visitor=%s&wait=%d", serverAddr, path, visitorName, timeout))
	ret.Payload = payloadType
	if err != nil { //http通信错误
		ret.Set(false, err.Error())
	} else {
		ret_bytes, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil { //读取错误
			ret.Set(false, err.Error())
		} else {
			err = json.Unmarshal(ret_bytes, &ret)
			if err != nil { //json解析错误
				ret.Set(false, err.Error()+"\n"+string(ret_bytes))
			}
		}
	}
	return ret
}
func HttpReadWithTimeoutWithName(path string, timeout int, payloadType interface{}, visitor_name string) (ret ReturnWithPayload) {
	resp, err := http.Get(fmt.Sprintf("%s/%s?visitor=%s&wait=%d", serverAddr, path, visitor_name, timeout))
	ret.Payload = payloadType
	if err != nil { //http通信错误
		ret.Set(false, err.Error())
	} else {
		ret_bytes, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil { //读取错误
			ret.Set(false, err.Error())
		} else {
			err = json.Unmarshal(ret_bytes, &ret)
			if err != nil { //json解析错误
				ret.Set(false, err.Error()+"\n"+string(ret_bytes))
			}
		}
	}
	return ret
}

//无阻塞读
func HttpRead(path string, payloadType interface{}) (ret ReturnWithPayload) {
	resp, err := http.Get(fmt.Sprintf("%s/%s?visitor=%s", serverAddr, path, visitorName))
	ret.Payload = payloadType
	if err != nil { //http通信错误
		ret.Set(false, err.Error())
	} else {
		ret_bytes, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		//holmes.Infof("%s", string(ret_bytes))
		if err != nil { //读取错误
			ret.Set(false, err.Error())
		} else {
			err = json.Unmarshal(ret_bytes, &ret)
			if err != nil { //json解析错误
				ret.Set(false, err.Error())
			}
		}
	}
	return ret
}

//写入
func HttpWrite(path string, input []byte) (ret ReturnBase) {
	buf := bytes.NewBuffer(input)
	resp, err := http.Post(fmt.Sprintf("%s/%s?visitor=%s", serverAddr, path, visitorName), "application/json", buf)

	if err != nil { //http通信错误
		ret.Set(false, err.Error())
		holmes.Errorln(err)
	} else {
		ret_bytes, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil { //读取错误
			ret.Set(false, err.Error())
		} else {
			err = json.Unmarshal(ret_bytes, &ret)
			if err != nil { //json解析错误
				ret.Set(false, err.Error())
			}
		}
	}
	return ret
}
