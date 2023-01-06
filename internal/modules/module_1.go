package modules

import (
	"fmt"
	"time"

	"github.com/DarmawanEfendi/scheduler/pkg/scheduler"
)

type Module1 struct {
	funcParams []string
}

func NewModule1() IModules {
	return &Module1{
		funcParams: []string{"time1", "time2"},
	}
}

func (m *Module1) Call(params map[string]interface{}) (interface{}, map[string]interface{}, error) {
	newParams, err := m.validateFuncParams(params)
	if err != nil {
		return nil, nil, err
	}
	return m.createFunc(), newParams, nil
}

func (m *Module1) validateFuncParams(params map[string]interface{}) (map[string]interface{}, error) {
	return validateFuncParams(params, m.funcParams)
}

func (*Module1) createFunc() func(taskParams *scheduler.TaskParams, userParams map[string]interface{}) {
	return func(taskParams *scheduler.TaskParams, userParams map[string]interface{}) {
		fmt.Printf("[module_1] State now: %v\n", taskParams.State)
		for i := 0; i < 70; i++ {
			taskParams.State["i"] = i
			select {
			case <-taskParams.Ctx.Done():
				fmt.Println("[module_1] Cancelling the task...")
				return
			default:
				fmt.Printf("[module_1] Engine do task 1: %v\n", userParams)
				time.Sleep(500 * time.Millisecond)
			}
		}
	}
}
