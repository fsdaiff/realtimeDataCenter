package component

//路径 函数,用于访问json时，递归
import (
	"strconv"
	"strings"
)

const (
	PathNodeString = iota
	PathNodeArray  = iota
)
const rootName = "根节点"

//是否为纯数字组成的 ，是true
func whetherNumberString(input string) bool {
	if len(input) == 0 {
		return false
	}
	for i := 0; i < len(input); i++ {
		//0-9
		if 0x30 > input[i] || input[i] > 0x39 {
			return false
		}
	}
	return true
}

//路径节点
type PathNode struct {
	whether_array bool
	value         string
	array         int
}

//节点类型
func (node PathNode) Type() int {
	if node.whether_array {
		return PathNodeArray
	}
	return PathNodeString
}
func (node PathNode) Number() int {
	return node.array
}

//Create(path string) (remain string, ok bool) //从路径中提取当前路径,返回剩余路径,如果包含数组则同时提取数组
//内容
func (node PathNode) Name() string {
	return node.value
}
func (node *PathNode) Input(name string) {
	node.value = name
	if whetherNumberString(name) {
		node.array, _ = strconv.Atoi(name)
		node.whether_array = true
	} else {
		node.array = -1
		node.whether_array = false
	}
}

//路径
type Path struct {
	nodes []PathNode
}

func (path Path) Iter(do func(input PathNode) bool) {
	if path.nodes == nil {
		return
	}

	length := len(path.nodes)

	for i := 0; i < length; i++ {
		if do(path.nodes[i]) {
			break
		}
	}
}

//导入
func (path *Path) InputFromString(name string) error {
	s := strings.Split(name, "/")
	path.nodes = make([]PathNode, 0)
	length := len(s)
	var a PathNode
	for i := 0; i < length; i++ {
		if s[i] != "" {
			a.Input(s[i])
			path.nodes = append(path.nodes, a)
		}
	}
	return nil
}

//输出 A/B/C
func (path Path) Output2String() string {
	var path_all string
	for i := 0; i < len(path.nodes); i++ {
		if i != 0 {
			path_all += "/" + path.nodes[i].Name()
		} else {
			path_all += path.nodes[i].Name()
		}

	}
	return path_all
}
func (node *Path) Father() Path {
	var a Path
	length := len(node.nodes)
	if length > 1 {
		a.nodes = node.nodes[:len(node.nodes)-1]
	} else {
		a.InputFromString("")
	}

	return a
}
func (node *Path) Last() PathNode {
	if node.nodes == nil {
		return PathNode{}
	}
	length := len(node.nodes)
	if length == 0 {
		return PathNode{}
	}
	return node.nodes[len(node.nodes)-1]
}
