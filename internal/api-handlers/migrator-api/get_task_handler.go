package apihandlers

import (
	"net/http"

	"github.com/DarmawanEfendi/scheduler/internal/api-handlers/httputil"
	"github.com/DarmawanEfendi/scheduler/pkg/scheduler"
	"github.com/gorilla/mux"
)

type GetTaskResponse struct {
	Id             string                 `json:"id"`
	Type           string                 `json:"type"`
	CronExpression string                 `json:"cron_expression,omitempty"`
	Status         string                 `json:"status"`
	State          map[string]interface{} `json:"state"`
	Module         struct {
		Name   string                 `json:"name"`
		Params map[string]interface{} `json:"params"`
	} `json:"module"`
}

type GetTaskHandler struct {
	scheduler *scheduler.Scheduler
}

func NewGetTaskHandler(sch *scheduler.Scheduler) http.Handler {
	handler := GetTaskHandler{
		scheduler: sch,
	}
	return &handler
}

func (h *GetTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		httputil.ReturnError(w, http.StatusBadRequest, "Id is missing in parameters")
		return
	}

	task, err := h.scheduler.GetTask(id)
	if err != nil {
		httputil.ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := GetTaskResponse{
		Id:             task.GetId(),
		Type:           task.GetType(),
		CronExpression: task.GetCronExpression(),
		Status:         task.GetStatus(),
		State:          task.GetState(),
		Module: struct {
			Name   string                 "json:\"name\""
			Params map[string]interface{} "json:\"params\""
		}{
			Name:   task.GetModuleName(),
			Params: task.GetModuleParams(),
		},
	}
	httputil.ReturnSuccess(w, http.StatusOK, response, nil)
}
