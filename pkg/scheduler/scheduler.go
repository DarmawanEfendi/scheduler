package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
)

// ScheduleTaskFileState as a standard format for load/store the schedule data
type ScheduleTaskFileState struct {
	Module struct {
		Name   string                 `json:"name"`
		Params map[string]interface{} `json:"params"`
	} `json:"module"`
	Task struct {
		Id               string                 `json:"id"`
		Type             string                 `json:"type"`
		CronExpression   string                 `json:"cron_expression"`
		IsImmediatelyRun bool                   `json:"is_immediately_run"`
		RemoveFile       bool                   `json:"-"`
		Status           string                 `json:"status"`
		State            map[string]interface{} `json:"state"`
	} `json:"task"`
}

// Scheduler type hold all fields about schedule
type Scheduler struct {
	scheduler  *gocron.Scheduler
	tasks      map[string]*Task
	tasksMutex sync.RWMutex
}

// Create new scheduler instance
func New() *Scheduler {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	s := gocron.NewScheduler(loc)

	// if we have state to load, we need to load here...

	return &Scheduler{
		scheduler:  s,
		tasks:      make(map[string]*Task),
		tasksMutex: sync.RWMutex{},
	}
}

// Unmarshall convert from byte into ScheduleTaskFileState
func (s *Scheduler) Unmarshall(bytes []byte) (ScheduleTaskFileState, error) {
	var scheduleTaskFileState ScheduleTaskFileState

	err := json.Unmarshal(bytes, &scheduleTaskFileState)
	if err != nil {
		return ScheduleTaskFileState{}, err
	}
	return scheduleTaskFileState, nil
}

// LoadState: Load state from bytes and create responding job and task
func (s *Scheduler) LoadState(fileState ScheduleTaskFileState, modFunc interface{}, modFuncParams map[string]interface{}) error {
	// only accept "Pause" state...

	if fileState.Task.Status != string(TaskPauseStatus) {
		return fmt.Errorf("task id: %v with status: %v not able to load state", fileState.Task.Id, fileState.Task.Status)
	}

	taskType := TaskOnceType
	if fileState.Task.Type == string(TaskRecurringType) {
		taskType = TaskRecurringType
	}

	newTask := NewTask(fileState.Task.Id, taskType, fileState.Module.Name, fileState.Module.Params, fileState.Task.CronExpression, modFunc, modFuncParams)
	newTask.SetPause()
	newTask.SetState(fileState.Task.State)
	s.setTask(fileState.Task.Id, newTask)

	if fileState.Task.IsImmediatelyRun {
		s.ResumeTask(newTask.id)
	}
	return nil
}

// StoreState: Store state to....
func (s *Scheduler) StoreState() ([]ScheduleTaskFileState, error) {
	// pause all jobs first if running....
	// all cancel job, remove file on gcs....

	result := make([]ScheduleTaskFileState, 0)
	// get snapshots
	allTasks := s.GetTasks()
	// allJobs := s.getJobs()

	for taskId, task := range allTasks {
		immediatelyRun := false
		removeFile := false
		haveJob := task.GetJob() != nil

		if haveJob || task.IsRunning() || task.IsResume() || (task.IsCompleted() && task.IsRecurring()) {
			_, err := s.PauseTask(taskId)
			if err != nil {
				return result, err
			}
			task.SetPause()
			immediatelyRun = true
		}

		removeFile = !haveJob && (task.IsCancel() || task.IsCompleted())

		result = append(result, ScheduleTaskFileState{
			Module: struct {
				Name   string                 "json:\"name\""
				Params map[string]interface{} "json:\"params\""
			}{
				Name:   task.module,
				Params: task.moduleParams,
			},
			Task: struct {
				Id               string                 "json:\"id\""
				Type             string                 "json:\"type\""
				CronExpression   string                 "json:\"cron_expression\""
				IsImmediatelyRun bool                   "json:\"is_immediately_run\""
				RemoveFile       bool                   "json:\"-\""
				Status           string                 "json:\"status\""
				State            map[string]interface{} "json:\"state\""
			}{
				Id:               task.id,
				Type:             string(task.taskType),
				Status:           string(task.taskStatus),
				CronExpression:   task.cronExp,
				IsImmediatelyRun: immediatelyRun,
				RemoveFile:       removeFile,
				State:            task.taskState,
			},
		})

	}

	return result, nil
}

// StartAsync: Start scheduler async without blocking the current process
func (s *Scheduler) StartAsync() {
	s.scheduler.WaitForScheduleAll()
	s.scheduler.StartAsync()
}

// Stop: Stop scheduler will wait all running tasks to finish before returning,
// so it is safe to assume that running jobs will finishe when calling this.
// No-op if scheduler is already stopped.
func (s *Scheduler) Stop() {
	s.scheduler.Stop()
}

// addTask: Add once/recurring task
func (s *Scheduler) addTask(taskId string, module string, moduleParams map[string]interface{}, cronExpression string, taskType TaskType, runImmediately bool,
	overrideTaskState map[string]interface{}, taskFunc TaskFunc, taskFuncParams map[string]interface{}) (*Task, error) {
	if taskId == "" {
		// this mean the task is new and need to generate new id
		taskId = uuid.New().String()
	}

	taskState := make(map[string]interface{})

	if overrideTaskState != nil {
		taskState = overrideTaskState
	}

	taskParams := &TaskParams{
		Ctx:   nil,
		State: taskState,
	}

	allParams := make([]interface{}, 0)
	allParams = append(allParams, taskParams)
	if len(taskFuncParams) > 0 {
		allParams = append(allParams, taskFuncParams)
	}

	var job *gocron.Job
	var err error

	if taskType == TaskOnceType {
		job, err = s.scheduler.
			Every(1).
			LimitRunsTo(1).
			SingletonMode().
			Tag(taskId).
			Do(taskFunc, allParams...)
	} else {
		job, err = s.scheduler.
			Cron(cronExpression).
			SingletonMode().
			Tag(taskId).
			Do(taskFunc, allParams...)
	}

	if err != nil {
		log.Printf("failed to add %v task: %v\n", taskType, err)
		return nil, err
	}

	job.SetEventListeners(s.beforeTaskExecution(taskId, taskParams), s.afterTaskExecution(taskId, taskParams))

	if runImmediately {
		s.scheduler.RunByTag(taskId)
	}

	newTask := NewTask(taskId, taskType, module, moduleParams, cronExpression, taskFunc, taskFuncParams)
	newTask.SetJob(job)
	s.setTask(taskId, newTask)

	log.Printf("success added %v task\n", taskType)

	return newTask, nil
}

func (s *Scheduler) getTask(taskId string) (*Task, error) {
	s.tasksMutex.RLock()
	defer s.tasksMutex.RUnlock()
	task, ok := s.tasks[taskId]
	if !ok {
		return nil, errors.New("task not found")
	}

	return task, nil
}

func (s *Scheduler) setTask(taskId string, task *Task) {
	s.tasksMutex.Lock()
	s.tasks[taskId] = task
	s.tasksMutex.Unlock()
}

// func (s *Scheduler) removeTask(taskId string) {
// 	s.tasksMutex.Lock()
// 	delete(s.tasks, taskId)
// 	s.tasksMutex.Unlock()
// }

func (s *Scheduler) removeJob(task *Task) {
	s.scheduler.RemoveByReference(task.job)
	task.SetJob(nil)
}

// beforeTaskExecution: event called before execution
func (s *Scheduler) beforeTaskExecution(taskId string, taskParams *TaskParams) func() {
	return func() {
		log.Printf("set running status on task: beforeTaskExecution\n")

		task, err := s.getTask(taskId)
		if err != nil {
			log.Printf("getTask error on before task execution: %v\n", err)
			return
		}

		task.mutex.Lock()
		defer task.mutex.Unlock()

		ctx, cancel := context.WithCancel(context.Background())
		taskParams.Ctx = ctx // set context for job as reference
		newTask := task.BeforeExecution(ctx, cancel)
		taskParams.State = newTask.GetState()

		s.setTask(taskId, task)
	}
}

// afterTaskExecution: event called after execution
func (s *Scheduler) afterTaskExecution(taskId string, taskParams *TaskParams) func() {
	return func() {
		task, err := s.getTask(taskId)
		if err != nil {
			log.Printf("getTask error on after task execution: %v\n", err)
			return
		}

		task.mutex.Lock()
		defer task.mutex.Unlock()
		newTask := task.AfterExecution(taskParams.State)
		taskParams.State = newTask.GetState()

		if newTask.IsCompleted() && !newTask.IsRecurring() {
			s.removeJob(newTask)
		}
		s.setTask(taskId, newTask)

		log.Printf("set completed status on task: afterTaskExecution: id: %v - type: %v - status: %v - state: %v\n", taskId, task.taskType, task.taskStatus, task.taskState)
	}
}

// AddOnceTask: Add new once task and immediately running the task.
func (s *Scheduler) AddOnceTask(module string, moduleParams map[string]interface{}, taskFunc TaskFunc, taskFuncParams map[string]interface{}) (*Task, error) {
	return s.addTask("", module, moduleParams, "", TaskOnceType, true, nil, taskFunc, taskFuncParams)
}

// AddRecurringTask: Add new recurring task and running based on the cron expression or schedule.
func (s *Scheduler) AddRecurringTask(module string, moduleParams map[string]interface{}, cronExpression string, taskFunc TaskFunc, taskFuncParams map[string]interface{}) (*Task, error) {
	return s.addTask("", module, moduleParams, cronExpression, TaskRecurringType, false, nil, taskFunc, taskFuncParams)
}

// CancelTask: Cancel the running task based on task-id, this action will not store the last state.
// OnceTask will cancel the running task, remove the job from scheduler.
// RecurringTask will cancel the running task, remove the job from scheduler and recreate the job.
func (s *Scheduler) CancelTask(taskId string, forceDelete bool) (*Task, error) {
	task, err := s.getTask(taskId)
	if err != nil {
		log.Printf("getTask error: %v\n", err)
		return nil, err
	}

	task.mutex.Lock()
	defer task.mutex.Unlock()

	if !task.CanCancel() {
		return nil, fmt.Errorf("Task %s failed to move cancel state", taskId)
	}

	task.SetCancel()

	s.removeJob(task)
	// s.removeTask(taskId)

	if !task.IsRecurring() || forceDelete {
		return task, nil
	}

	// only add new on recurring
	newTask, err := s.addTask(taskId, task.module, task.moduleParams, task.cronExp, task.taskType, false, nil, task.taskFunc, task.taskFuncParams)
	if err != nil {
		log.Printf("failed to add recurring on cancel task: %v", err)
	}

	return newTask, nil
}

// PauseTask: Pause the running task based on task-id, this action will store the last state.
// OnceTask will cancel the running task, store the state, remove the job from scheduler.
// RecurringTask will cancel the running task, store the state, remove the job from scheduler.
func (s *Scheduler) PauseTask(taskId string) (*Task, error) {
	task, err := s.getTask(taskId)
	if err != nil {
		log.Printf("getTask error: %v\n", err)
		return nil, err
	}

	task.mutex.Lock()

	if !task.CanPause() {
		return nil, fmt.Errorf("Task %s failed to move pause state", taskId)
	}

	if task.IsResume() || task.IsNew() {
		// if resume, the task stil waiting time to run, so we need to force add this line.
		task.taskReachFinalStep = true
	}
	task.SetPause()
	task.mutex.Unlock()

	s.removeJob(task)

	for !task.GetReachFinalStep() {
		log.Printf("waiting until task on final step...\n")
		time.Sleep(500 * time.Millisecond)
	}

	s.setTask(taskId, task)

	log.Printf("success pausing task...\n")

	return task, nil
}

// ResumeTask: Resume the pause task based on task-id, and will take the last state.
// OnceTask & RecurringTask will add new job, running immediately based on last state.
func (s *Scheduler) ResumeTask(taskId string) (*Task, error) {
	task, err := s.getTask(taskId)
	if err != nil {
		log.Printf("getTask error: %v\n", err)
		return nil, err
	}

	task.mutex.Lock()
	defer task.mutex.Unlock()

	if !task.CanResume() {
		return nil, fmt.Errorf("Task %s failed to move resume state", taskId)
	}

	runImmediately := true
	if task.IsRecurring() {
		runImmediately = false
	}

	newTask, err := s.addTask(taskId, task.module, task.moduleParams, task.cronExp, task.taskType, runImmediately, task.taskState, task.taskFunc, task.taskFuncParams)
	if err != nil {
		log.Printf("failed to add %v task on resume task: %v", task.taskType, err)
	}

	newTask.mutex.Lock()
	defer newTask.mutex.Unlock()
	newTask.SetState(task.taskState)
	newTask.SetResume()
	s.setTask(taskId, newTask)

	return newTask, nil
}

// func (s *Scheduler) getJobs() map[string]*gocron.Job {
// 	s.jobsMutex.Lock()
// 	allJobs := make(map[string]*gocron.Job)
// 	for key, value := range s.jobs {
// 		allJobs[key] = value
// 	}
// 	defer s.jobsMutex.Unlock()
// 	return allJobs
// }

// GetTasks: Get all registered tasks
func (s *Scheduler) GetTasks() map[string]*Task {
	s.tasksMutex.Lock()
	newTasks := make(map[string]*Task)
	for key, value := range s.tasks {
		newTasks[key] = value
	}
	defer s.tasksMutex.Unlock()
	return newTasks
}

// GetTask: Get task based on id
func (s *Scheduler) GetTask(taskId string) (*Task, error) {
	task, err := s.getTask(taskId)
	if err != nil {
		log.Printf("getTask error: %v\n", err)
		return nil, err
	}
	return task, nil
}
