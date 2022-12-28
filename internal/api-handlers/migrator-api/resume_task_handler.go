package apihandlers

import (
	"net/http"

	"github.com/DarmawanEfendi/scheduler/internal/api-handlers/httputil"
	"github.com/DarmawanEfendi/scheduler/pkg/scheduler"
	"github.com/gorilla/mux"
)

type ResumeTaskResponse struct {
	Id             string `json:"id"`
	Type           string `json:"type"`
	CronExpression string `json:"cron_expression,omitempty"`
	Status         string `json:"status"`
}

type ResumeTaskHandler struct {
	scheduler *scheduler.Scheduler
}

func NewResumeTaskHandler(sch *scheduler.Scheduler) http.Handler {
	handler := ResumeTaskHandler{
		scheduler: sch,
	}
	return &handler
}

func (h *ResumeTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		httputil.ReturnError(w, http.StatusBadRequest, "Id is missing in parameters")
		return
	}

	task, err := h.scheduler.ResumeTask(id)
	if err != nil {
		httputil.ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := ResumeTaskResponse{
		Id:             task.GetId(),
		Type:           task.GetType(),
		CronExpression: task.GetCronExpression(),
		Status:         task.GetStatus(),
	}
	httputil.ReturnSuccess(w, http.StatusOK, response, nil)
}
