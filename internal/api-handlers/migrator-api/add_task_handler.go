package apihandlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/DarmawanEfendi/scheduler/internal/api-handlers/httputil"
	"github.com/DarmawanEfendi/scheduler/internal/modules"
	"github.com/DarmawanEfendi/scheduler/pkg/scheduler"
)

type AddTaskRequest struct {
	Module struct {
		Name   string                 `json:"name"`
		Params map[string]interface{} `json:"params"`
	} `json:"module"`
	Task struct {
		Type           string `json:"type"`
		CronExpression string `json:"cron_expression"`
	} `json:"task"`
}

type AddTaskResponse struct {
	Id             string `json:"id"`
	Type           string `json:"type"`
	CronExpression string `json:"cron_expression,omitempty"`
	Status         string `json:"status"`
}

type AddTaskHandler struct {
	scheduler *scheduler.Scheduler
}

func NewAddTaskHandler(sch *scheduler.Scheduler) http.Handler {
	handler := AddTaskHandler{
		scheduler: sch,
	}
	return &handler
}

func (h *AddTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body AddTaskRequest
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httputil.ReturnError(w, http.StatusBadRequest, "Failed to read body")
		return
	}

	err = json.Unmarshal(reqBody, &body)
	if err != nil {
		httputil.ReturnError(w, http.StatusBadRequest, "Failed to parse body")
		return
	}

	if body.Module.Name == "" {
		httputil.ReturnError(w, http.StatusBadRequest, "Module name is required")
		return
	}

	module, err := modules.Get(body.Module.Name)
	if err != nil {
		httputil.ReturnError(w, http.StatusBadRequest, err.Error())
		return
	}

	response := AddTaskResponse{
		Type:           body.Task.Type,
		CronExpression: body.Task.CronExpression,
	}

	modFunc, modFuncParams, err := module.Call(body.Module.Params)
	if err != nil {
		httputil.ReturnError(w, http.StatusBadRequest, err.Error())
		return
	}

	switch body.Task.Type {
	case strings.ToUpper(string(scheduler.TaskOnceType)):
		{
			task, err := h.scheduler.AddOnceTask(body.Module.Name, body.Module.Params, modFunc, modFuncParams)
			if err != nil {
				httputil.ReturnError(w, http.StatusBadRequest, "Failed adding task...")
				return
			}
			response.Id = task.GetId()
			response.Status = task.GetStatus()
			break
		}
	case strings.ToUpper(string(scheduler.TaskRecurringType)):
		{
			if body.Task.CronExpression == "" {
				httputil.ReturnError(w, http.StatusBadRequest, "Cron Expression is required when task type is recurring")
				return
			}

			task, err := h.scheduler.AddRecurringTask(body.Module.Name, body.Module.Params, body.Task.CronExpression, modFunc, modFuncParams)

			if err != nil {
				httputil.ReturnError(w, http.StatusBadRequest, "Failed adding task...")
			}
			response.Id = task.GetId()
			response.Status = task.GetStatus()
			break
		}
	default:
		{
			httputil.ReturnError(w, http.StatusBadRequest, "No task type available")
			return
		}
	}
	httputil.ReturnSuccess(w, http.StatusCreated, response, nil)
}
