package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	migratorApiHandlers "github.com/DarmawanEfendi/scheduler/internal/api-handlers/migrator-api"
	"github.com/DarmawanEfendi/scheduler/internal/gcs"
	"github.com/DarmawanEfendi/scheduler/internal/modules"
	"github.com/DarmawanEfendi/scheduler/pkg/scheduler"
	"github.com/gorilla/mux"
)

// The following constants define all possible exit code returned by the server
const (
	CodeSuccess = iota
	CodeBadConfig
	codeFailServe
)

// Server is the long-running application with graceful shutdown support.
type Server struct {
	httpServer   *http.Server
	scheduler    *scheduler.Scheduler
	gcsClient    gcs.IGCS
	waitShutdown chan bool
}

// New returns a Server with all configurations read from the config file.
func New() (*Server, error) {
	s := &Server{
		waitShutdown: make(chan bool),
	}
	r := mux.NewRouter()

	// ==============================================================
	// Modules
	modules.Register("module_1", modules.NewModule1())
	modules.Register("module_2", modules.NewModule2())

	// ==============================================================
	// GCS
	gcs, err := gcs.New(true)
	if err != nil {
		return nil, err
	}
	s.gcsClient = gcs

	//  ==============================================================
	// Load GCS
	taskFiles, err := s.gcsClient.ReadAllObjects("tasks", -1)
	if err != nil {
		log.Printf("error on load gcs: %v\n", err)
		return nil, err
	}

	//  ==============================================================
	// Scheduler
	scheduler := scheduler.New()
	s.scheduler = scheduler

	err = s.LoadScheduleFileState(taskFiles)
	if err != nil {
		return nil, err
	}

	homeHandler := migratorApiHandlers.NewHomeHandler()
	r.HandleFunc("/", homeHandler.ServeHTTP)

	addTaskHandler := migratorApiHandlers.NewAddTaskHandler(s.scheduler)
	r.HandleFunc("/add-task", addTaskHandler.ServeHTTP) // new add task

	getTasksHandler := migratorApiHandlers.NewGetTasksHandler(s.scheduler)
	r.HandleFunc("/tasks", getTasksHandler.ServeHTTP) // Get all tasks

	getTaskHandler := migratorApiHandlers.NewGetTaskHandler(s.scheduler)
	r.HandleFunc("/tasks/{id}", getTaskHandler.ServeHTTP) // get task detail

	pauseTaskHandler := migratorApiHandlers.NewPauseTaskHandler(s.scheduler)
	r.HandleFunc("/tasks/{id}/pause", pauseTaskHandler.ServeHTTP) // pause task

	resumeTaskHandler := migratorApiHandlers.NewResumeTaskHandler(s.scheduler)
	r.HandleFunc("/tasks/{id}/resume", resumeTaskHandler.ServeHTTP) // resume task

	cancelTaskHandler := migratorApiHandlers.NewCancelTaskHandler(s.scheduler)
	r.HandleFunc("/tasks/{id}/cancel", cancelTaskHandler.ServeHTTP) // cancel task

	taskHistoriesHandler := migratorApiHandlers.NewGetTaskHistoryHandler(s.scheduler, s.gcsClient)
	r.HandleFunc("/task-histories", taskHistoriesHandler.ServeHTTP) // get task histories

	// ==============================================================
	// HTTP Server.
	{
		server := &http.Server{
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			Addr:         "0.0.0.0:8000",
			Handler:      r,
		}
		server.RegisterOnShutdown(func() {
			// close all open client
			log.Print("On Shutdown....\n")
			_ = s.stopClient()
			s.waitShutdown <- true
		})
		s.httpServer = server
	}

	return s, nil
}

func (s *Server) LoadScheduleFileState(taskFiles map[string][]byte) error {
	for _, fileContent := range taskFiles {
		fileState, err := s.scheduler.Unmarshall(fileContent)
		if err != nil {
			return err
		}

		module, err := modules.Get(fileState.Module.Name)
		if err != nil {
			return err
		}

		modFunc, modFuncParams, err := module.Call(fileState.Module.Params)
		if err != nil {
			return err
		}

		err = s.scheduler.LoadState(fileState, modFunc, modFuncParams)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) StoreScheduleFileState() error {
	log.Print("Getting all task states....\n")
	states, err := s.scheduler.StoreState()
	if err != nil {
		return err
	}

	log.Print("Stopping scheduler....\n")
	s.scheduler.Stop()

	for _, state := range states {
		taskBytes, err := json.Marshal(state)
		if err != nil {
			return err
		}

		log.Printf("Uploading file task: %v.json ....\n", state.Task.Id)
		err = s.gcsClient.UploadObject("tasks", fmt.Sprintf("%v.json", state.Task.Id), taskBytes)
		if err != nil {
			return err
		}

		if state.Task.RemoveFile {
			log.Printf("Moving file task: %v.json to histories....\n", state.Task.Id)
			err = s.gcsClient.MoveObject("tasks", "task-histories", fmt.Sprintf("%v.json", state.Task.Id))
			if err != nil {
				return err
			}
			continue
		}
	}

	return nil
}

// Start will run when the service is started.
func (s *Server) Start() int {

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Print("Scheduler Started....")
	s.scheduler.StartAsync()

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v\n", err)
		}
	}()
	log.Print("Server Started....")

	<-done
	log.Print("Server Stopped....")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown failed: %v", err)
	}

	<-s.waitShutdown
	return s.Shutdown()
}

func (s *Server) stopClient() error {
	log.Print("Store all scheduler jobs....\n")
	err := s.StoreScheduleFileState()
	if err != nil {
		log.Printf("error on store all scheduler job on shutdown : %v\n", err)
	}
	log.Print("Stopping gcs....\n")
	s.gcsClient.Close()
	return nil
}

// Shutdown will run when the service is stopped.
func (s *Server) Shutdown() int {
	return CodeSuccess
}

// Run creates a server
func Run() int {
	s, err := New()
	if err != nil {
		log.Printf("error on bad config: %v\n", err)
		return CodeBadConfig
	}

	return s.Start()
}
