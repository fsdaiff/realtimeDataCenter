package component

import (
	"encoding/json"
	"testing"
)

func Test_singleNodePerformance(t *testing.T) {
	var a singleNodePerformance
	ret, err := json.Marshal(&a)
	if err != nil {
		t.Error(err)
	} else {
		t.Logf("%s", string(ret))
	}

}
