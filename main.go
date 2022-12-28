package main

import (
	"fmt"
	"time"

	"github.com/DarmawanEfendi/scheduler/internal/modules"
	"github.com/DarmawanEfendi/scheduler/pkg/scheduler"
)

func main() {
	s := scheduler.New()
	s.StartAsync()
	defer s.Stop()

	mod1 := modules.NewModule1()
	params1 := map[string]interface{}{
		"time1": 100,
		"time2": 200,
	}
	modFunc1, modFunc1Params, err := mod1.Call(params1)
	if err != nil {
		fmt.Printf("Error call module 1: %v", err)
	}

	_, err = s.AddOnceTask("module_1", params1, modFunc1, modFunc1Params)
	if err != nil {
		fmt.Printf("Error adding once task 1: %v", err)
	}

	mod2 := modules.NewModule2()
	modFunc2, modFunc2Params, err := mod2.Call(map[string]interface{}{
		"time1": 100,
	})
	if err != nil {
		fmt.Printf("Error call module 2: %v", err)
	}

	_, err = s.AddOnceTask("module_2", make(map[string]interface{}), modFunc2, modFunc2Params)

	if err != nil {
		fmt.Printf("Error adding once task 2: %v", err)
	}

	mod3 := modules.NewModule2()
	modFunc3, modFunc3Params, err := mod3.Call(map[string]interface{}{
		"time1": 100,
	})
	if err != nil {
		fmt.Printf("Error call module 3: %v", err)
	}

	task3, err := s.AddRecurringTask("module_2", make(map[string]interface{}), "*/1 * * * *", modFunc3, modFunc3Params)

	if err != nil {
		fmt.Printf("Error adding once task 3: %v", err)
	}

	fmt.Printf("All tasks: %v\n", s.GetTasks())

	time.Sleep(60 * time.Second)
	fmt.Println("Pausing task...")
	s.PauseTask(task3.GetId())

	time.Sleep(5 * time.Second)
	s.ResumeTask(task3.GetId())

	time.Sleep(60 * time.Second)
	fmt.Println("Cancelling first task...")
	s.CancelTask(task3.GetId(), false)

	time.Sleep(3 * time.Minute)
	fmt.Println("Cancelling second task...")
	s.CancelTask(task3.GetId(), false)

	fmt.Printf("All tasks: %v\n", s.GetTasks())
	time.Sleep(10 * time.Minute)

}
