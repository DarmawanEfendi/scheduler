package modules

import (
	"errors"

	"github.com/DarmawanEfendi/scheduler/pkg/scheduler"
)

type IModules interface {
	Call(params map[string]interface{}) (interface{}, map[string]interface{}, error) // func, params
	createFunc() func(taskParams *scheduler.TaskParams, userParams map[string]interface{})
	validateFuncParams(params map[string]interface{}) (map[string]interface{}, error)
}

type Modules map[string]IModules

var modules Modules

func Register(name string, module IModules) error {
	if modules == nil {
		modules = make(Modules)
	}

	if _, ok := modules[name]; !ok {
		modules[name] = module
		return nil
	}
	return errors.New("module already registered")
}

func Get(name string) (IModules, error) {
	if module, ok := modules[name]; ok {
		return module, nil
	}
	return nil, errors.New("module not registered")
}
