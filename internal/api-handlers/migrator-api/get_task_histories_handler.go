package apihandlers

import (
	"log"
	"net/http"

	"github.com/DarmawanEfendi/scheduler/internal/api-handlers/httputil"
	"github.com/DarmawanEfendi/scheduler/internal/gcs"
	"github.com/DarmawanEfendi/scheduler/pkg/scheduler"
)

type GetTaskHistoryResponse struct {
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

type GetTaskHistoryHandler struct {
	scheduler *scheduler.Scheduler
	gcsClient gcs.IGCS
}

func NewGetTaskHistoryHandler(sch *scheduler.Scheduler, gcs gcs.IGCS) http.Handler {
	handler := GetTaskHistoryHandler{
		scheduler: sch,
		gcsClient: gcs,
	}
	return &handler
}

func (h *GetTaskHistoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	taskFiles, err := h.gcsClient.ReadAllObjects("task-histories", 10)
	if err != nil {
		log.Printf("error on load gcs: %v\n", err)
		httputil.ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]GetTaskHistoryResponse, 0)
	for _, taskFile := range taskFiles {
		taskState, err := h.scheduler.Unmarshall(taskFile)
		if err != nil {
			httputil.ReturnError(w, http.StatusInternalServerError, err.Error())
			return
		}

		response = append(response, GetTaskHistoryResponse{
			Id:             taskState.Task.Id,
			Type:           taskState.Task.Type,
			CronExpression: taskState.Task.CronExpression,
			Status:         taskState.Task.Status,
			State:          taskState.Task.State,
			Module: struct {
				Name   string                 "json:\"name\""
				Params map[string]interface{} "json:\"params\""
			}{
				Name:   taskState.Module.Name,
				Params: taskState.Module.Params,
			},
		})
	}

	httputil.ReturnSuccess(w, http.StatusOK, response, nil)
}
