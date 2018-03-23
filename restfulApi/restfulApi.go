package rtCenterApi

//注册restfulApi
//访问数据

import (
	"encoding/json"
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-swagger12"
	"github.com/fsdaiff/logClient"
	"github.com/fsdaiff/realtimeDataCenter/component"
	"io/ioutil"
	"net/http"
	"strconv"
)

const defaultVisitorName = "默认访问者"

type Restful struct {
	read        component.ControlRead
	write       component.ControlWrite
	wsContainer *restful.Container
}

func (api *Restful) Register(addr string, read component.ControlRead, write component.ControlWrite) {
	//输入检查
	if read == nil || write == nil {
		holmes.Fatalln("api不能为空")
	}
	//注册
	api.read = read
	api.write = write

	api.wsContainer = restful.NewContainer()
	api.wsContainer.Router(restful.CurlyRouter{})

	ws2 := new(restful.WebService)
	ws2.Path("/performance/").
		Consumes(restful.MIME_XML, restful.MIME_JSON).
		Produces(restful.MIME_JSON, restful.MIME_XML)
	ws2.Route(ws2.GET("").To(api.performance))
	api.wsContainer.Add(ws2)

	ws := new(restful.WebService)

	ws.Path("/v1.0/realtime/").
		Consumes(restful.MIME_XML, restful.MIME_JSON).
		Produces(restful.MIME_JSON, restful.MIME_XML) // you can specify this per route as well
	ws.Route(ws.GET("").To(api.visitorRead).
		// docs
		Doc("读所有数据").
		Param(ws.QueryParameter("visitor", "访问者名称").DataType("string")).
		Param(ws.QueryParameter("wait", "等待超时时间,如果没有该参数，则立即返回").DataType("string")))
	ws.Route(ws.GET("/{sub_path:*}").To(api.visitorRead).
		// docs
		Doc("读数据").
		Param(ws.PathParameter("sub_path", "访问路径").DataType("string")).
		Param(ws.QueryParameter("visitor", "访问者名称").DataType("string")).
		Param(ws.QueryParameter("wait", "等待超时时间,如果没有该参数，则立即返回").DataType("string")))
	ws.Route(ws.POST("").To(api.visitorWrite).
		// docs
		Doc("写所有数据").
		Param(ws.QueryParameter("visitor", "访问者名称").DataType("string")))
	ws.Route(ws.POST("/{sub_path:*}").To(api.visitorWrite).
		// docs
		Doc("写数据").
		Param(ws.PathParameter("sub_path", "访问路径").DataType("string").DefaultValue("")).
		Param(ws.QueryParameter("visitor", "访问者名称").DataType("string")))
	ws.ApiVersion("v1.0")
	api.wsContainer.Add(ws)

	// You can install the Swagger Service which provides a nice Web UI on your REST API
	// You need to download the Swagger HTML5 assets and change the FilePath location in the config below.
	// Open http://localhost:8080/apidocs and enter http://localhost:8080/apidocs.json in the api input field.
	config := swagger.Config{
		WebServices:    api.wsContainer.RegisteredWebServices(), // you control what services are visible
		WebServicesUrl: addr,
		ApiPath:        "/apidocs.json",

		// Optionally, specify where the UI is located
		SwaggerPath:     "/apidocs/",
		SwaggerFilePath: "/Users/emicklei/xProjects/swagger-ui/dist"}
	swagger.RegisterSwaggerService(config, api.wsContainer)

	holmes.Eventf("实时数据汇总进程restfulapi开始监听%s", addr)
	server := &http.Server{Addr: addr, Handler: api.wsContainer}
	err := server.ListenAndServe()
	if err != nil {
		holmes.Fatalf("restful 监听错误退出%s", err.Error())

	}
}
func (api *Restful) performance(request *restful.Request, response *restful.Response) {
	if api.read == nil {
		response.Write([]byte("核心未注册performance"))
	} else {
		response.Write([]byte(api.read.PerformanceShow()))
	}
}
func (api *Restful) visitorRead(request *restful.Request, response *restful.Response) {
	sub_path := request.PathParameter("sub_path")
	wait := request.QueryParameter("wait")
	visitor_name := request.QueryParameter("visitor")
	if visitor_name == "" {
		visitor_name = defaultVisitorName
		holmes.Errorf("访问者名称为空,采用%s", visitor_name)
	}
	holmes.Debugf("sub_path:%s visitor:%s wait:%s", sub_path, visitor_name, wait)
	var ret ReturnWithPayloadOnReadBuffer
	if wait == "" {

		ret.Payload = api.read.ReadQuickly(visitor_name, sub_path)
		if ret.Payload.Nil() {
			ret.Set(false, "读返回nil")
		} else {
			ret.Set(true, "")
		}
	} else {
		wait_time, err := strconv.Atoi(wait)
		if err != nil {
			ret.Set(false, err.Error())
		} else {
			ret.Set(true, "")
		}
		ret.SetPayload(api.read.ReadWithTime(visitor_name, sub_path, wait_time))
		if ret.Payload.Nil() {
			ret.Set(false, "读返回nil")
		} else {
			ret.Set(true, "")
		}
	}
	response.Write([]byte(ret.String()))
}
func (api *Restful) visitorWrite(request *restful.Request, response *restful.Response) {
	var ret ReturnBase
	defer request.Request.Body.Close()
	datas, err := ioutil.ReadAll(request.Request.Body)

	if err == nil {
		sub_path := request.PathParameter("sub_path")
		visitor_name := request.QueryParameter("visitor")
		if visitor_name == "" {
			visitor_name = defaultVisitorName
			holmes.Errorf("访问者名称为空,采用%s", visitor_name)
		}
		var data_input interface{}
		err = json.Unmarshal(datas, &data_input)
		if err == nil {
			ret.Set(true, "")
			err = api.write.WriteQuickly(visitor_name, sub_path, data_input)
			if err != nil {
				ret.Set(false, err.Error())
			} else {
				ret.Set(true, "")
			}
		} else {
			ret.Set(false, err.Error())
		}

	} else {
		ret.Set(false, err.Error())
	}

	response.WriteAsJson(ret)
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

type ReturnWithPayload struct {
	ReturnBase
	Payload component.ReadResult `json:"data"`
}

func (ret ReturnWithPayload) String() string {
	if ret.Payload.Nil() && ret.Reason == "ok" {
		ret.Set(false, "数据为空")
	}
	if ret.Payload.Nil() {
		ret.Payload = "null"
	}
	return fmt.Sprintf(`{"status":"%s","reason":"%s","data":%s}`, ret.Status, ret.Reason, ret.Payload)

}

type ReturnWithPayloadOnReadBuffer struct {
	ReturnBase
	Writer    string               `json:"visitor"`
	Timestamp int64                `json:"timestamp"`
	Payload   component.ReadResult `json:"data"`
}

func (ret *ReturnWithPayloadOnReadBuffer) SetPayload(result component.ReadBufferResult) {
	if result.Nil() {
		ret.Set(false, "无效操作")
	} else {
		ret.Set(true, "")
		ret.Writer, ret.Payload, ret.Timestamp = result.Get()
	}
}
func (ret ReturnWithPayloadOnReadBuffer) String() string {
	if ret.Payload.Nil() && ret.Reason == "ok" {
		ret.Set(false, "数据为空")
	}
	if ret.Payload.Nil() {
		ret.Payload = "null"
	}
	return fmt.Sprintf(`{"status":"%s","reason":"%s","visitor":%s,"data":%s}`, ret.Status, ret.Reason, ret.Writer, ret.Payload)

}
