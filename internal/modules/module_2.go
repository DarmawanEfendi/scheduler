package modules

import (
	"fmt"
	"time"

	"github.com/DarmawanEfendi/scheduler/pkg/scheduler"
)

type Module2 struct {
	funcParams []string
}

func NewModule2() IModules {
	return &Module2{
		funcParams: []string{"time1"},
	}
}

func (m *Module2) Call(params map[string]interface{}) (interface{}, map[string]interface{}, error) {
	newParams, err := m.validateFuncParams(params)
	if err != nil {
		return nil, nil, err
	}
	return m.createFunc(), newParams, nil
}

func (m *Module2) validateFuncParams(params map[string]interface{}) (map[string]interface{}, error) {
	return validateFuncParams(params, m.funcParams)
}

func (*Module2) createFunc() interface{} {
	return func(taskParams *scheduler.TaskParams, userParams map[string]interface{}) {
		fmt.Printf("[module_2] State now: %v\n", taskParams.State)
		for i := 0; i < 100; i++ {
			taskParams.State["i"] = i
			select {
			case <-taskParams.Ctx.Done():
				fmt.Println("[module_2] Cancelling the task...")
				return
			default:
				fmt.Printf("[module_2] Engine do task: %v\n", userParams)
				time.Sleep(500 * time.Millisecond)
			}
		}
	}
}
