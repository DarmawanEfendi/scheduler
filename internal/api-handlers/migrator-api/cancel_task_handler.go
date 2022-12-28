package apihandlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/DarmawanEfendi/scheduler/internal/api-handlers/httputil"
	"github.com/DarmawanEfendi/scheduler/pkg/scheduler"
	"github.com/gorilla/mux"
)

type CancelTaskResponse struct {
	Id             string `json:"id"`
	Type           string `json:"type"`
	CronExpression string `json:"cron_expression,omitempty"`
	Status         string `json:"status"`
}

type CancelTaskRequest struct {
	IsForceDelete bool `json:"is_force_delete"`
}

type CancelTaskHandler struct {
	scheduler *scheduler.Scheduler
}

func NewCancelTaskHandler(sch *scheduler.Scheduler) http.Handler {
	handler := CancelTaskHandler{
		scheduler: sch,
	}
	return &handler
}

func (h *CancelTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		httputil.ReturnError(w, http.StatusBadRequest, "Id is missing in parameters")
		return
	}

	var body CancelTaskRequest
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

	task, err := h.scheduler.CancelTask(id, body.IsForceDelete)
	if err != nil {
		httputil.ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := CancelTaskResponse{
		Id:             task.GetId(),
		Type:           task.GetType(),
		CronExpression: task.GetCronExpression(),
		Status:         task.GetStatus(),
	}
	httputil.ReturnSuccess(w, http.StatusOK, response, nil)
}
