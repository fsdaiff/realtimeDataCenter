package component

import (
	"fmt"
	"testing"
)

func Test_WhetherContain(t *testing.T) {
	a := "hello1"
	b := "hello11"
	c := "hello2"
	t.Logf("%s %s %s", a, b, c)
	if WhetherContain(a, b) {
		t.Logf("%s 想关于 %s的判断正确", a, b)
	} else {
		t.Errorf("%s 不相关于 %s的判断错误", a, b)
	}
	if WhetherContain(a, c) {
		t.Errorf("%s 想关于 %s的判断错误", a, c)
	} else {
		t.Logf("%s 不相关于 %s的判断正确", a, c)
	}
	if WhetherContain("", a) {
		t.Logf("%s 想关于 根节点的判断正确", a)
	} else {
		t.Errorf("%s 不相关于 根节点的判断错误", a)
	}
	if WhetherContain("", "") {
		t.Logf("根节点 想关于 根节点的判断正确")
	} else {
		t.Errorf("根节点 不相关于 根节点的判断错误")
	}
}
func Test_longPollReadControl(t *testing.T) {
	var a longPollReadControl
	length := 4
	t.Logf("开始添加")
	for i := 0; i < length; i++ {
		path := fmt.Sprintf("a%d/b%d/c%d", i, i, i)
		a.applyFor(CreateVisitorRwait(*newReadRequest("dgl", path, 1)))
		path = fmt.Sprintf("a%d/b%d", i, i)
		a.applyFor(CreateVisitorRwait(*newReadRequest("dgl", path, 1)))
	}

	t.Logf("添加%v", a.allKeys())

	t.Logf("测试相关性")
	affect := []string{"a1/", "a1/b1/", "a1/b1/c1", "a1/b1/c1/d1"}
	for i := 0; i < len(affect); i++ {
		affected := a.Affect(affect[i])
		if len(affected) == 2 {
			t.Logf("%s相关于%v判断正确", affect[i], affected)
		} else {
			t.Errorf("%s相关于%v判断错误", affect[i], affected)
		}
	}

	t.Logf("测试非相关性")
	not_affect := []string{"a2/b1"}
	for i := 0; i < len(not_affect); i++ {
		affected := a.Affect(not_affect[i])
		if len(affected) == 0 {
			t.Logf("%s没有相关判断正确", not_affect[i])
		} else {
			t.Errorf("%s对%v判断错误", not_affect[i], affected)
		}
	}
}
