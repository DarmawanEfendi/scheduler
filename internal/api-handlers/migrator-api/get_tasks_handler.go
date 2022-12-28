package apihandlers

import (
	"net/http"

	"github.com/DarmawanEfendi/scheduler/internal/api-handlers/httputil"
	"github.com/DarmawanEfendi/scheduler/pkg/scheduler"
)

type GetTasksResponse struct {
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

type GetTasksHandler struct {
	scheduler *scheduler.Scheduler
}

func NewGetTasksHandler(sch *scheduler.Scheduler) http.Handler {
	handler := GetTasksHandler{
		scheduler: sch,
	}
	return &handler
}

func (h *GetTasksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	tasks := h.scheduler.GetTasks()

	response := make([]GetTasksResponse, 0)

	for key, val := range tasks {
		response = append(response, GetTasksResponse{
			Id:             key,
			Type:           val.GetType(),
			CronExpression: val.GetCronExpression(),
			Status:         val.GetStatus(),
			State:          val.GetState(),
			Module: struct {
				Name   string                 "json:\"name\""
				Params map[string]interface{} "json:\"params\""
			}{
				Name:   val.GetModuleName(),
				Params: val.GetModuleParams(),
			},
		})
	}

	httputil.ReturnSuccess(w, http.StatusOK, response, nil)
}
