package component

//树数据:写和读
import (
	"encoding/json"
	"errors"
	"fmt"
	//"github.com/json-iterator/go"
	"github.com/fsdaiff/logClient"
	"reflect"
	"strconv"
)

const (
	objectType = iota
	arrayType
	valueType
)

//用于递归的
type Node struct {
	JsonObject
	rootType int //节点类型
	name     string
}

func (object *Node) SetValue(name string, value interface{}) {
	//类型
	switch value.(type) {
	case map[string]interface{}: //正常节点
		object.rootType = objectType
	case []interface{}: //数组节点
		object.rootType = arrayType
	default: //叶节点
		object.rootType = valueType

	}
	object.name = name
	object.JsonObject.SetValue(value)
}
func (object *Node) IsDir() bool {
	if object.rootType != valueType {
		return true
	}
	return false
}

//获取子节点
func (object *Node) Map() map[string]interface{} {
	switch object.rootType {
	case objectType:
		var ret map[string]interface{} = make(map[string]interface{})
		var values map[string]interface{} = object.value.(map[string]interface{})
		for key, value := range values {
			ret[fmt.Sprintf("%s/%s", object.name, key)] = value
		}
		return ret
	case arrayType:
		var ret map[string]interface{} = make(map[string]interface{})
		ptr := object.value.([]interface{})
		for i := 0; i < len(ptr); i++ {
			ret[fmt.Sprintf("%s/%d", object.name, i)] = ptr[i]
		}
		return ret
	case valueType:
		return nil
	}
	return nil
}
func IterAllPath(road string, input interface{}) []string {
	var a Node
	a.SetValue(road, input)

	if a.IsDir() {
		var ret []string = make([]string, 1)
		//添加自己
		ret[0] = road
		//添加子节点
		recursion := a.Map()
		for key, value := range recursion {
			result := IterAllPath(key, value)
			ret = append(ret, result...)
		}
		return ret
	} else {
		return []string{road}
	}
}

type JsonObject struct {
	value interface{}
}

func (object *JsonObject) SetValue(value interface{}) {
	object.value = value
}

//写时，不改变键，只改变键值
//返回键的地址，这样键值被改变，也不影响
func (object *JsonObject) Find(path Path) (data interface{}) {

	//指针
	var ptr interface{}
	var ok bool
	ptr = object.value
	//
	do := func(input PathNode) bool {
		switch input.Type() {
		case PathNodeString:
			switch ptr.(type) {
			case map[string]interface{}: //正常节点
				if ptr, ok = ptr.(map[string]interface{})[input.Name()]; ok {
					return false
				} else {
					ptr = nil
					return true //错误
				}
			case []interface{}: //数组节点
				ptr = nil
				return true //错误
			default: //叶节点
				return true
			}
		case PathNodeArray:
			switch ptr.(type) {
			case map[string]interface{}: //正常节点
				ptr = nil
				return true //错误
			case []interface{}: //数组节点
				if len(ptr.([]interface{})) > input.Number() {
					ptr = ptr.([]interface{})[input.Number()]
					return false
				} else {
					ptr = nil
					return true //错误
				}
			default: //叶节点
				ptr = nil
				return true //错误
			}
			return false //错误
		default:
			ptr = nil
			return true //错误
		}
		return true
	}
	path.Iter(do)
	return ptr
}

func (object *JsonObject) Write(visitor VisitorWrite) error {
	path := visitor.GetPath()
	father_node := path.Father()
	last_node := path.Last()
	ptr := object.Find(father_node) //找到要写入元素的父节点
	data := visitor.GetData()
	if ptr == nil {
		return errors.New("未发现:" + father_node.Output2String())
	}
	//要写入的父节点类型
	switch ptr.(type) {
	case map[string]interface{}: //正常节点
		//要写入节点的类型
		switch last_node.Type() {
		case PathNodeString:
			if _, ok := ptr.(map[string]interface{})[last_node.Name()]; ok {
				//是否需要类型保护 ?
				if reflect.TypeOf(ptr.(map[string]interface{})[last_node.Name()]) == reflect.TypeOf(data) {
					ptr.(map[string]interface{})[last_node.Name()] = data
					holmes.Debugf("%s向%s写入%v", visitor.Name(), visitor.GetPath().Output2String(), data)
				} else {
					return errors.New(path.Output2String() + "写入类型" + reflect.TypeOf(data).String() + "与实际类型" + reflect.TypeOf(ptr.(map[string]interface{})[last_node.Name()]).String() + "不相同")
				}

			} else {
				if path.Output2String() != "" {
					//str, _ := jsoniter.MarshalToString(ptr)
					return errors.New("节点" + father_node.Output2String() + "中不存在子节点" + last_node.Name() + ",实际:" + path.Output2String())
				} else {
					//写入所有数据
					ptr = data
					holmes.Debugf("%s向覆盖所有数据%v", visitor.GetPath().Output2String(), data)
				}
			}
		case PathNodeArray:
			return errors.New(last_node.Name() + ",类型错误（不为数组序号）")
		default:
			return errors.New(last_node.Name() + ",未知节点类型")
		}
	case []interface{}: //数组节点
		switch last_node.Type() {
		case PathNodeString:
			return errors.New(last_node.Name() + ",类型错误（不为字典类型）")
		case PathNodeArray:
			if len(ptr.([]interface{})) > last_node.Number() {
				ptr.([]interface{})[last_node.Number()] = data
			} else {
				//实际长度
				true_length := strconv.FormatInt(int64(len(ptr.([]interface{}))), 10)
				return errors.New("节点" + father_node.Output2String() + ",数组越界" + last_node.Name() + ",实际大小:" + true_length)
			}
		default:
			return errors.New(last_node.Name() + ",未知节点类型")
		}
	default: //叶节点
		switch last_node.Type() {
		case PathNodeString:
			return errors.New(last_node.Name() + ",类型错误（不为字典类型）")
		case PathNodeArray:
			return errors.New(last_node.Name() + ",类型错误（不为数组序号）")
		default:
			return errors.New(last_node.Name() + ",类型错误(不为叶节点)")
		}
	}
	return nil
}

//如果未找到就返回空字符串
func (object *JsonObject) Read(visitor Visitor) ReadResult {
	//holmes.Infof("%s向%s读", visitor.Name(), visitor.GetPath().Output2String())
	ptr := object.Find(visitor.GetPath())
	if ptr == nil {
		return ""
	}
	json_bytes, err := json.Marshal(ptr)
	if err != nil {
		holmes.Errorln(err)
	}
	return ReadResult(json_bytes)
}
