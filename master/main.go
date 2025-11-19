package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/juandjimenezjdjv/Proy_2_SO/common"
)

// Master es el coordinador principal del sistema distribuido
type Master struct {
	id     string
	config *common.Config
	logger *common.Logger

	// Almacenamiento en memoria
	workers map[string]*common.WorkerInfo
	jobs    map[string]*common.Job
	tasks   map[string]*common.Task

	// Sincronización
	workersMutex sync.RWMutex
	jobsMutex    sync.RWMutex
	tasksMutex   sync.RWMutex

	// Control
	shutdownChan chan bool
}

// NewMaster crea una nueva instancia del Master
func NewMaster(config *common.Config) *Master {
	return &Master{
		id:           fmt.Sprintf("master-%d", time.Now().Unix()),
		config:       config,
		logger:       common.NewLogger("MASTER", config.LogLevel),
		workers:      make(map[string]*common.WorkerInfo),
		jobs:         make(map[string]*common.Job),
		tasks:        make(map[string]*common.Task),
		shutdownChan: make(chan bool),
	}
}

// Start inicia el servidor HTTP del master
func (m *Master) Start() error {
	m.logger.Info("Iniciando Master en %s:%d", m.config.MasterHost, m.config.MasterPort)

	// Configurar rutas HTTP
	http.HandleFunc("/api/v1/workers/register", m.handleRegisterWorker)
	http.HandleFunc("/api/v1/workers/heartbeat", m.handleHeartbeat)
	http.HandleFunc("/api/v1/jobs", m.handleJobs)
	http.HandleFunc("/api/v1/jobs/", m.handleJobDetails)
	http.HandleFunc("/api/v1/tasks/update", m.handleTaskUpdate)
	http.HandleFunc("/health", m.handleHealth)

	// Iniciar monitor de heartbeats
	go m.monitorWorkerHeartbeats()

	// Iniciar servidor HTTP
	addr := fmt.Sprintf("%s:%d", m.config.MasterHost, m.config.MasterPort)
	m.logger.Info("Master escuchando en http://%s", addr)

	return http.ListenAndServe(addr, nil)
}

// handleRegisterWorker maneja el registro de un nuevo worker
func (m *Master) handleRegisterWorker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req common.RegisterWorkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.logger.Error("Error decodificando registro de worker: %v", err)
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	m.logger.Info("Registrando worker: %s desde %s", req.WorkerID, req.Address)

	// Registrar worker
	m.workersMutex.Lock()
	m.workers[req.WorkerID] = &common.WorkerInfo{
		ID:            req.WorkerID,
		Address:       req.Address,
		Status:        common.WorkerStatusUp,
		RegisteredAt:  time.Now(),
		LastHeartbeat: time.Now(),
		ActiveTasks:   0,
		TotalTasks:    0,
	}
	m.workersMutex.Unlock()

	// Responder
	resp := common.RegisterWorkerResponse{
		Success:      true,
		Message:      "Worker registrado exitosamente",
		MasterID:     m.id,
		HeartbeatSec: m.config.HeartbeatSec,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleHeartbeat maneja los heartbeats de los workers
func (m *Master) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req common.HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.logger.Error("Error decodificando heartbeat: %v", err)
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	// Actualizar heartbeat
	m.workersMutex.Lock()
	if worker, exists := m.workers[req.WorkerID]; exists {
		worker.LastHeartbeat = time.Now()
		worker.ActiveTasks = req.ActiveTasks
		worker.Status = common.WorkerStatusUp
		m.logger.Debug("Heartbeat recibido de worker %s (tareas activas: %d)", req.WorkerID, req.ActiveTasks)
	} else {
		m.logger.Warn("Heartbeat de worker desconocido: %s", req.WorkerID)
	}
	m.workersMutex.Unlock()

	// Responder
	resp := common.HeartbeatResponse{
		Success:   true,
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleJobs maneja las peticiones de jobs (GET lista, POST crear)
func (m *Master) handleJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		m.handleListJobs(w, r)
	case http.MethodPost:
		m.handleCreateJob(w, r)
	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

// handleListJobs lista todos los jobs
func (m *Master) handleListJobs(w http.ResponseWriter, r *http.Request) {
	m.jobsMutex.RLock()
	jobsList := make([]*common.Job, 0, len(m.jobs))
	for _, job := range m.jobs {
		jobsList = append(jobsList, job)
	}
	m.jobsMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobsList)
}

// handleCreateJob crea un nuevo job
func (m *Master) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var job common.Job
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		m.logger.Error("Error decodificando job: %v", err)
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	// Generar ID y configurar job
	job.ID = fmt.Sprintf("job-%d", time.Now().UnixNano())
	job.Status = common.JobStatusAccepted
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()
	job.Progress = 0.0

	m.logger.Info("Creando nuevo job: %s (nombre: %s)", job.ID, job.Name)

	// Guardar job
	m.jobsMutex.Lock()
	m.jobs[job.ID] = &job
	m.jobsMutex.Unlock()

	// Responder
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(job)
}

// handleJobDetails obtiene detalles de un job específico
func (m *Master) handleJobDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	// Extraer ID del job de la URL (formato: /api/v1/jobs/{id})
	jobID := r.URL.Path[len("/api/v1/jobs/"):]

	m.jobsMutex.RLock()
	job, exists := m.jobs[jobID]
	m.jobsMutex.RUnlock()

	if !exists {
		http.Error(w, "Job no encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

// handleTaskUpdate maneja actualizaciones de estado de tareas desde workers
func (m *Master) handleTaskUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req common.TaskUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.logger.Error("Error decodificando actualización de tarea: %v", err)
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	m.logger.Info("Actualización de tarea %s: %s", req.TaskID, req.Status)

	// Actualizar tarea
	m.tasksMutex.Lock()
	if task, exists := m.tasks[req.TaskID]; exists {
		task.Status = req.Status
		if req.Status == common.TaskStatusFailed {
			task.Error = req.Error
		}
		now := time.Now()
		if req.Status == common.TaskStatusCompleted {
			task.CompletedAt = &now
		}
	}
	m.tasksMutex.Unlock()

	// Responder
	resp := common.TaskUpdateResponse{
		Success: true,
		Message: "Tarea actualizada",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleHealth verifica la salud del master
func (m *Master) handleHealth(w http.ResponseWriter, r *http.Request) {
	m.workersMutex.RLock()
	workersUp := 0
	for _, worker := range m.workers {
		if worker.Status == common.WorkerStatusUp {
			workersUp++
		}
	}
	totalWorkers := len(m.workers)
	m.workersMutex.RUnlock()

	health := map[string]interface{}{
		"status":        "healthy",
		"master_id":     m.id,
		"workers_up":    workersUp,
		"workers_total": totalWorkers,
		"timestamp":     time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// monitorWorkerHeartbeats monitorea los heartbeats de los workers
func (m *Master) monitorWorkerHeartbeats() {
	ticker := time.NewTicker(time.Duration(m.config.HeartbeatSec) * time.Second)
	defer ticker.Stop()

	timeoutDuration := time.Duration(m.config.HeartbeatTimeoutSec) * time.Second

	for {
		select {
		case <-ticker.C:
			m.workersMutex.Lock()
			now := time.Now()
			for _, worker := range m.workers {
				if now.Sub(worker.LastHeartbeat) > timeoutDuration {
					if worker.Status == common.WorkerStatusUp {
						m.logger.Warn("Worker %s no responde (último heartbeat: %v)", worker.ID, worker.LastHeartbeat)
						worker.Status = common.WorkerStatusDown
					}
				}
			}
			m.workersMutex.Unlock()
		case <-m.shutdownChan:
			m.logger.Info("Deteniendo monitor de heartbeats")
			return
		}
	}
}

// Shutdown detiene el master de forma ordenada
func (m *Master) Shutdown() {
	m.logger.Info("Iniciando apagado del Master...")
	close(m.shutdownChan)
}

func main() {
	config := common.LoadConfig()
	master := NewMaster(config)

	if err := master.Start(); err != nil {
		master.logger.Error("Error iniciando Master: %v", err)
	}
}
