package apihandlers

import (
	"net/http"

	"github.com/DarmawanEfendi/scheduler/internal/api-handlers/httputil"
	"github.com/DarmawanEfendi/scheduler/pkg/scheduler"
	"github.com/gorilla/mux"
)

type PauseTaskResponse struct {
	Id             string                 `json:"id"`
	Type           string                 `json:"type"`
	CronExpression string                 `json:"cron_expression,omitempty"`
	Status         string                 `json:"status"`
	State          map[string]interface{} `json:"state"`
}

type PauseTaskHandler struct {
	scheduler *scheduler.Scheduler
}

func NewPauseTaskHandler(sch *scheduler.Scheduler) http.Handler {
	handler := PauseTaskHandler{
		scheduler: sch,
	}
	return &handler
}

func (h *PauseTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		httputil.ReturnError(w, http.StatusBadRequest, "Id is missing in parameters")
		return
	}

	task, err := h.scheduler.PauseTask(id)
	if err != nil {
		httputil.ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := PauseTaskResponse{
		Id:             task.GetId(),
		Type:           task.GetType(),
		CronExpression: task.GetCronExpression(),
		Status:         task.GetStatus(),
		State:          task.GetState(),
	}
	httputil.ReturnSuccess(w, http.StatusOK, response, nil)
}
