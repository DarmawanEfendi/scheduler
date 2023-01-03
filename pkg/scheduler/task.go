package scheduler

import (
	"context"
	"sync"

	"github.com/go-co-op/gocron"
)

type TaskStatus string
type TaskType string
type TaskFunc interface{}

const (
	TaskNewStatus       TaskStatus = "New"
	TaskRunningStatus   TaskStatus = "Running"
	TaskPauseStatus     TaskStatus = "Pause" // Resume->Running Once: Remove the job: Run immediately, Recurring: Remove the job; Run as scheduled
	TaskResumeStatus    TaskStatus = "Resume"
	TaskCancelStatus    TaskStatus = "Cancel" // Once: Cancel , Remove the job, recurring: cancel -> remove the current job and create new recurringhv
	TaskCompletedStatus TaskStatus = "Completed"

	TaskOnceType      TaskType = "Once"
	TaskRecurringType TaskType = "Recurring"
)

type TaskParams struct {
	Ctx   context.Context
	State map[string]interface{}
}

type Task struct {
	id      string
	cronExp string

	taskType           TaskType
	taskStatus         TaskStatus
	taskFunc           TaskFunc
	taskFuncParams     map[string]interface{}
	taskState          map[string]interface{}
	taskReachFinalStep bool

	ctx    context.Context
	cancel context.CancelFunc
	mutex  sync.Mutex

	module       string
	moduleParams map[string]interface{}

	job *gocron.Job
}

func NewTask(taskId string, taskType TaskType, module string, moduleParams map[string]interface{},
	cronExpression string, taskFunc TaskFunc, taskFuncParams map[string]interface{}) *Task {
	return &Task{
		id:                 taskId,
		cronExp:            cronExpression,
		taskType:           taskType,
		taskStatus:         TaskNewStatus,
		taskFunc:           taskFunc,
		taskFuncParams:     taskFuncParams,
		taskState:          make(map[string]interface{}),
		taskReachFinalStep: false,

		ctx:    nil,
		cancel: nil,
		mutex:  sync.Mutex{},

		module:       module,
		moduleParams: moduleParams,

		job: nil,
	}
}

func (t *Task) BeforeExecution(ctx context.Context, cancel context.CancelFunc) *Task {
	t.ctx = ctx
	t.cancel = cancel
	t.SetRunning()
	t.taskReachFinalStep = false
	return t
}

func (t *Task) AfterExecution(taskState map[string]interface{}) *Task {
	t.SetState(taskState)
	t.taskReachFinalStep = true
	if t.IsRunning() {
		t.SetCompleted()
		t.SetState(make(map[string]interface{}))
	}

	// this comming from cancel task and task is recurring
	if t.IsNew() && t.IsRecurring() {
		t.SetState(make(map[string]interface{}))
	}

	if t.IsPause() {
		t.SetState(taskState)
	}

	if t.IsCancel() {
		t.SetState(make(map[string]interface{}))
	}

	return t
}

func (t *Task) GetReachFinalStep() bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.taskReachFinalStep
}

func (t *Task) SetState(vals map[string]interface{}) {
	t.taskState = vals
}

func (t *Task) GetState() map[string]interface{} {
	return t.taskState
}

func (t *Task) SetJob(job *gocron.Job) {
	t.job = job
}

func (t *Task) GetJob() *gocron.Job {
	return t.job
}

func (t *Task) SetRunning() {
	t.taskStatus = TaskRunningStatus
}

func (t *Task) SetResume() {
	t.taskStatus = TaskResumeStatus
}

func (t *Task) SetCompleted() {
	t.taskStatus = TaskCompletedStatus
	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
		t.ctx = nil
	}
}

func (t *Task) SetCancel() {
	t.taskStatus = TaskCancelStatus
	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
		t.ctx = nil
	}
}

func (t *Task) SetPause() {
	t.taskStatus = TaskPauseStatus
	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
		t.ctx = nil
	}
}

func (t *Task) IsPause() bool {
	return t.taskStatus == TaskPauseStatus
}

func (t *Task) IsRunning() bool {
	return t.taskStatus == TaskRunningStatus
}

func (t *Task) IsResume() bool {
	return t.taskStatus == TaskResumeStatus
}

func (t *Task) IsNew() bool {
	return t.taskStatus == TaskNewStatus
}

func (t *Task) IsCompleted() bool {
	return t.taskStatus == TaskCompletedStatus
}

func (t *Task) IsCancel() bool {
	return t.taskStatus == TaskCancelStatus
}

func (t *Task) IsRecurring() bool {
	return t.taskType == TaskRecurringType
}

func (t *Task) GetId() string {
	return t.id
}

func (t *Task) GetStatus() string {
	return string(t.taskStatus)
}

func (t *Task) GetType() string {
	return string(t.taskType)
}

func (t *Task) GetCronExpression() string {
	return t.cronExp
}

func (t *Task) GetModuleName() string {
	return t.module
}

func (t *Task) GetModuleParams() map[string]interface{} {
	return t.moduleParams
}

func (t *Task) CanResume() bool {
	return t.taskStatus == TaskPauseStatus && t.job == nil
}

func (t *Task) CanPause() bool {
	return t.job != nil && t.taskStatus != TaskPauseStatus
}

func (t *Task) CanCancel() bool {
	return t.job != nil && t.taskStatus != TaskCancelStatus
}
