package component

import (
	//"encoding/json"
	"github.com/json-iterator/go"
	"testing"
)

var (
	test_json = `{
	"errno": 0,
	"errmsg": "成功",
	"data": {
		"array":[1,2,3,4],
		"array_map":[
			{"config":"hello"}
		],
		"config_list": {
			"reg_conf": {
				"s": {
					"ss": "s1",
					"m": {
						"mm": "mm",
						"j": {
							"leng": 22,
							"sf": {
								"gf": "ss"
							}
						}
					}
				}
			}
		}
	}
	}`
	test_json2 = `{"a":1,"b":{"c":2,"d":"3"}}`
)
var (
	true_input  = make(map[string]interface{})
	true_input2 = make(map[string]interface{})
	map_3x3     = make(map[string]interface{})
	map_4x3     = make(map[string]interface{})
	map_5x3     = make(map[string]interface{})
	map_6x3     = make(map[string]interface{})
	map_7x3     = make(map[string]interface{})
	//find_3x3    = CreateBtreeRoad2leaf(3)
)

//比较两个json是否一样(数据长度),一样返回true
func compare(a map[string]interface{}, b map[string]interface{}) bool {
	if a == nil || b == nil {
		return false
	}
	data1, _ := jsoniter.Marshal(a)
	data2, _ := jsoniter.Marshal(b)
	if len(data1) == len(data2) {
		return true
	}
	return false
}

//测试路径类
func Test_PathClass(t *testing.T) {
	var a Path
	var input string = "data/array_map/0/config"
	a.InputFromString(input)
	t.Logf("输入:%s", input)
	if a.Output2String() != input {
		t.Errorf("输入输出不相等")
	}
	t.Logf("输出:%s", a.Output2String())

	//轮训
	pool := func(input PathNode) bool {
		t.Logf("名称:%s,性质:%d", input.Name(), input.Type())
		return false
	}
	a.Iter(pool)
}

//测试从json中添加
func Test_InputFromJson(t *testing.T) {
	jsoniter.Unmarshal([]byte(test_json), &true_input)
	jsoniter.Unmarshal([]byte(test_json2), &true_input2)
	switch true_input["data"].(map[string]interface{})["array_map"].(type) {
	case map[string]interface{}: //正常节点
		t.Errorf("正常节点")
	case []map[string]interface{}:
		t.Errorf("数组map节点")
	case []interface{}: //数组节点
		t.Logf("数组节点")
	default: //叶节点
		t.Errorf("叶节点")
	}
	switch true_input["errno"].(type) {
	case map[string]interface{}: //正常节点
		t.Errorf("正常节点")
	case []map[string]interface{}:
		t.Errorf("数组map节点")
	case []interface{}: //数组节点
		t.Errorf("数组节点")
	default: //叶节点
		t.Logf("叶节点")
	}
}

func Test_Find(t *testing.T) {
	jsoniter.Unmarshal([]byte(test_json), &true_input)
	var a JsonObject
	var b Path
	a.value = true_input
	//寻找节点
	b.InputFromString("/data/array_map/0/config/")
	ptr := a.Find(b)
	str, err := jsoniter.MarshalToString(ptr)
	if err != nil {
		t.Errorf("%s", err.Error())
	} else {
		t.Logf("%s", str)
	}

	t.Logf("测试查找所有节点")
	b.InputFromString("")
	ptr = a.Find(b)
	str, err = jsoniter.MarshalToString(ptr)
	if err != nil {
		t.Errorf("%s", err.Error())
	} else {
		t.Logf("%s", str)
	}
}

func Test_WriteMap(t *testing.T) {
	jsoniter.Unmarshal([]byte(test_json), &true_input)
	var a JsonObject
	a.value = true_input
	var path string = "data/config_list/"
	b := new(VisitorW)
	b.Set("dgl", path)
	b.SaveData(true_input2)
	a.Write(b)

	result := a.Find(b.GetPath())
	if !compare(true_input2, result.(map[string]interface{})) {
		t.Errorf("向%s写入%v不成功%s", path, true_input2, result.(map[string]interface{}))
	} else {
		str, _ := jsoniter.MarshalToString(a.value)
		t.Logf("写入成功 %s", str)
	}
}

func Test_WriteArray(t *testing.T) {
	jsoniter.Unmarshal([]byte(test_json), &true_input)
	var a JsonObject
	a.value = true_input
	var path string = "data/array_map/0"
	b := new(VisitorW)
	b.Set("dgl", path)
	b.SaveData(true_input2)
	err := a.Write(b)
	if err != nil {
		t.Error(err)
	} else {
		str, _ := jsoniter.Marshal(a.value)
		t.Logf("写入后:%s", str)
		result := a.Find(b.GetPath())

		if !compare(true_input2, result.(map[string]interface{})) {
			t.Errorf("向%s写入%v不成功%s", path, true_input2, result.(map[string]interface{}))
		} else {
			str, _ := jsoniter.MarshalToString(a.value)
			t.Logf("写入成功 %s", str)
		}
	}
}
func Test_写数组越界检查(t *testing.T) {
	jsoniter.Unmarshal([]byte(test_json), &true_input)
	var a JsonObject
	a.value = true_input
	var path string = "data/array_map/1"
	b := new(VisitorW)
	b.Set("dgl", path)
	b.SaveData(true_input2)
	err := a.Write(b)
	if err != nil {
		t.Log(err)
	} else {
		t.Errorf("越界检查不通过")
	}
}

func Test_写叶子节点(t *testing.T) {
	jsoniter.Unmarshal([]byte(test_json), &true_input)
	var a JsonObject
	a.value = true_input
	var path string = "data/array_map/0/config"
	b := new(VisitorW)
	b.Set("dgl", path)
	b.SaveData("hello")
	err := a.Write(b)
	if err != nil {
		t.Error(err)
		str, _ := jsoniter.MarshalToString(a.value)
		t.Logf("%s", str)
	} else {
		result := a.Find(b.GetPath())
		if result.(string) == "hello" {
			t.Logf("写叶子节点成功")
		} else {
			t.Errorf("写入不成功")
		}
	}
}
func Test_read(t *testing.T) {
	jsoniter.Unmarshal([]byte(test_json), &true_input)
	var a JsonObject
	a.value = true_input

	var path [4]Path
	var path_string [4]string = [4]string{"data/config_list", "data/array_map/", "data/array_map/0", "data/array_map/0/config"}
	all, _ := jsoniter.Marshal(true_input)
	t.Logf("数据为:%s", all)
	for i := 0; i < len(path); i++ {
		path[i].InputFromString(path_string[i])
		t.Logf("查找:%s", path[i].Output2String())
		visitor := new(VisitorRwait)
		visitor.Set("dgl", path[i].Output2String())
		result := a.Read(visitor)
		if result == "" {
			t.Errorf("未发现:%s", path[i].Output2String())
		} else {
			str, err := jsoniter.MarshalToString(result)
			if err != nil {
				t.Errorf("解析错误%s", err.Error())
			} else {
				t.Logf("发现:%s", str)
			}
		}
	}
}
func Test_readNotExit(t *testing.T) {
	jsoniter.Unmarshal([]byte(test_json), &true_input)
	var a JsonObject
	a.value = true_input

	var path [4]Path
	var path_string [4]string = [4]string{"data/config_list1", "data/array_map1/", "data/array_map/1", "data/array_map/0/config1"}
	all, _ := jsoniter.Marshal(true_input)
	t.Logf("数据为:%s", all)
	for i := 0; i < len(path); i++ {
		path[i].InputFromString(path_string[i])
		t.Logf("查找:%s", path[i].Output2String())
		visitor := new(VisitorRwait)
		visitor.Set("dgl", path[i].Output2String())
		result := a.Read(visitor)
		if result == "" {
			t.Logf("未发现:%s", path[i].Output2String())
		} else {
			t.Errorf("不存在该路径，但找到了该值")
		}
	}
}

//测试遍历
func Test_recursion(t *testing.T) {
	var a JsonObject
	a.SetValue(true_input)
	ret := IterAllPath("", true_input)
	t.Logf("节点数%d", len(ret))
	t.Logf("%v", ret)
	for i := 0; i < len(ret); i++ {
		var b Path
		b.InputFromString(ret[i])
		result := a.Find(b)
		if result == nil {
			t.Errorf("没有%s", ret[i])
		}

	}
}

func Test_writeWithErrorType(t *testing.T) {
	var a JsonObject
	var c int = 0
	a.SetValue(true_input)
	var b *VisitorW = new(VisitorW)
	b.SaveData(c)
	b.Set("dgl", "errmsg")

	t.Logf("%s类型为string，写入int", b.GetPath().Output2String())
	err := a.Write(b)
	if err == nil {
		t.Errorf("类型校验未通过")
	} else {
		t.Logf("类型校验通过,%s", err.Error())
	}
}
