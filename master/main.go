package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/juandjimenezjdjv/Proy_2_SO/common"
)

// Master es el coordinador principal del sistema distribuido
type Master struct {
	id        string
	config    *common.Config
	logger    *common.Logger
	scheduler *Scheduler

	// Almacenamiento en memoria
	workers map[string]*common.WorkerInfo
	jobs    map[string]*common.Job
	tasks   map[string]*common.Task

	// Persistencia
	stateManager *common.StateManager

	// Sincronización
	workersMutex sync.RWMutex
	jobsMutex    sync.RWMutex
	tasksMutex   sync.RWMutex

	// Control
	shutdownChan chan bool
}

// NewMaster crea una nueva instancia del Master
func NewMaster(config *common.Config) *Master {
	m := &Master{
		id:           fmt.Sprintf("master-%d", time.Now().Unix()),
		config:       config,
		logger:       common.NewLogger("MASTER", config.LogLevel),
		workers:      make(map[string]*common.WorkerInfo),
		jobs:         make(map[string]*common.Job),
		tasks:        make(map[string]*common.Task),
		shutdownChan: make(chan bool),
	}
	m.scheduler = NewScheduler(m)

	// Inicializar persistencia
	m.stateManager = common.NewStateManager("./storage", m.logger, true)

	// Cargar estado previo si existe
	if snapshot, err := m.stateManager.LoadLatestState(); err != nil {
		m.logger.Warn("Error cargando estado previo: %v", err)
	} else if snapshot != nil {
		m.jobs = snapshot.Jobs
		m.tasks = snapshot.Tasks
		// No restaurar workers ya que deben re-registrarse
		m.logger.Info("Estado restaurado: %d jobs, %d tasks", len(m.jobs), len(m.tasks))
	}

	return m
}

// Start inicia el servidor HTTP del master
func (m *Master) Start() error {
	m.logger.Info("Iniciando Master en %s:%d", m.config.MasterHost, m.config.MasterPort)

	// Configurar rutas HTTP
	http.HandleFunc("/api/v1/workers/register", m.handleRegisterWorker)
	http.HandleFunc("/api/v1/workers/heartbeat", m.handleHeartbeat)
	http.HandleFunc("/api/v1/workers/tasks", m.handleGetTasks)
	http.HandleFunc("/api/v1/jobs", m.handleJobs)
	http.HandleFunc("/api/v1/jobs/", m.handleJobDetails)
	http.HandleFunc("/api/v1/tasks/update", m.handleTaskUpdate)
	http.HandleFunc("/health", m.handleHealth)

	// Iniciar monitor de heartbeats
	go m.monitorWorkerHeartbeats()

	// Iniciar auto-save de estado
	m.stateManager.StartAutoSave(30*time.Second, func() error {
		m.workersMutex.RLock()
		workers := m.workers
		m.workersMutex.RUnlock()

		m.jobsMutex.RLock()
		jobs := m.jobs
		m.jobsMutex.RUnlock()

		m.tasksMutex.RLock()
		tasks := m.tasks
		m.tasksMutex.RUnlock()

		return m.stateManager.SaveState(jobs, tasks, workers)
	})

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
		worker.Metrics = req.Metrics

		metricsStr := ""
		if req.Metrics != nil {
			metricsStr = fmt.Sprintf(" - %s", req.Metrics.FormatMetrics())
		}
		m.logger.Debug("Heartbeat recibido de worker %s (tareas activas: %d)%s", req.WorkerID, req.ActiveTasks, metricsStr)
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

// handleGetTasks permite a un worker obtener las tareas que se le han asignado
func (m *Master) handleGetTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	workerID := r.URL.Query().Get("worker_id")
	if workerID == "" {
		http.Error(w, "worker_id requerido", http.StatusBadRequest)
		return
	}

	// Buscar tareas asignadas a este worker
	m.tasksMutex.RLock()
	var assignedTasks []*common.Task
	for _, task := range m.tasks {
		if task.WorkerID == workerID && task.Status == common.TaskAssigned {
			assignedTasks = append(assignedTasks, task)
		}
	}
	m.tasksMutex.RUnlock()

	m.logger.Debug("Worker %s solicitó tareas: encontradas %d", workerID, len(assignedTasks))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assignedTasks)
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

// convertStagesToDAG convierte el formato de stages a DAG
func (m *Master) convertStagesToDAG(stages []JobRequestStage) common.DAG {
	nodes := make([]common.DAGNode, 0, len(stages))
	edges := make([]common.DAGEdge, 0)

	for _, stage := range stages {
		// Convertir el stage a un nodo DAG
		node := common.DAGNode{
			ID:       stage.StageID,
			Operator: common.OperatorType(stage.Operator),
		}

		// Extraer configuración común
		if inputPath, ok := stage.Config["input_path"].(string); ok {
			node.InputPaths = []string{inputPath}
		}
		if outputPath, ok := stage.Config["output_path"].(string); ok {
			node.OutputPath = outputPath
		}
		if partitions, ok := stage.Config["partitions"].(float64); ok {
			node.Partitions = int(partitions)
		}

		// Guardar toda la config en Params
		node.Params = stage.Config

		nodes = append(nodes, node)

		// Crear aristas desde las dependencias hacia este nodo
		for _, dep := range stage.Dependencies {
			edge := common.DAGEdge{
				From: dep,
				To:   stage.StageID,
			}
			edges = append(edges, edge)
		}
	}

	return common.DAG{
		Nodes: nodes,
		Edges: edges,
	}
}

// JobRequestStage representa un stage en el formato de request JSON
type JobRequestStage struct {
	StageID      string                 `json:"stage_id"`
	Operator     string                 `json:"operator"`
	Config       map[string]interface{} `json:"config"`
	Dependencies []string               `json:"dependencies"`
}

// JobRequest representa el formato de request para crear un job
type JobRequest struct {
	JobID  string            `json:"job_id"`
	Name   string            `json:"name"`
	Stages []JobRequestStage `json:"stages"`
}

// handleCreateJob crea un nuevo job
func (m *Master) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	// Leer el body una sola vez
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		m.logger.Error("Error leyendo body: %v", err)
		http.Error(w, "Error leyendo request", http.StatusBadRequest)
		return
	}

	// Intentar parsear como Job directo (con DAG)
	var jobDirect common.Job
	var dag common.DAG
	var jobID, jobName string

	if err := json.Unmarshal(bodyBytes, &jobDirect); err == nil && len(jobDirect.DAG.Nodes) > 0 {
		// Formato DAG directo
		dag = jobDirect.DAG
		jobID = jobDirect.ID
		jobName = jobDirect.Name
		m.logger.Info("Parseado como formato DAG directo (%d nodes)", len(dag.Nodes))
	} else {
		// Intentar parsear como JobRequest (formato con stages)
		var jobReq JobRequest
		if err := json.Unmarshal(bodyBytes, &jobReq); err != nil {
			m.logger.Error("Error decodificando job: %v", err)
			http.Error(w, "JSON inválido", http.StatusBadRequest)
			return
		}
		// Convertir stages a DAG
		dag = m.convertStagesToDAG(jobReq.Stages)
		jobID = jobReq.JobID
		jobName = jobReq.Name
		m.logger.Info("Parseado como formato stages (%d stages)", len(jobReq.Stages))
	}

	// Crear el job real
	job := common.Job{
		ID:          jobID,
		Name:        jobName,
		DAG:         dag,
		Status:      common.JobStatusAccepted,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		SubmittedAt: time.Now(),
		Progress:    0.0,
	}

	// Generar ID si no viene en el request
	if job.ID == "" {
		job.ID = fmt.Sprintf("job-%d", time.Now().UnixNano())
	}

	m.logger.Info("Creando nuevo job: %s (nombre: %s)", job.ID, job.Name)

	// Guardar job
	m.jobsMutex.Lock()
	m.jobs[job.ID] = &job
	m.jobsMutex.Unlock()

	// Programar job automáticamente
	go func() {
		m.logger.Info("Iniciando scheduling para job: %s", job.ID)
		if err := m.scheduler.ScheduleJob(&job); err != nil {
			m.logger.Error("Error scheduling job %s: %v", job.ID, err)
			m.jobsMutex.Lock()
			if j, exists := m.jobs[job.ID]; exists {
				j.Status = common.JobStatusFailed
				j.UpdatedAt = time.Now()
			}
			m.jobsMutex.Unlock()
		}
	}()

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
	var jobID string
	if task, exists := m.tasks[req.TaskID]; exists {
		task.Status = req.Status
		jobID = task.JobID
		if req.Status == common.TaskStatusFailed {
			task.Error = req.Error
		}
		if req.DurationMs > 0 {
			task.DurationMs = req.DurationMs
		}
		now := time.Now()
		if req.Status == common.TaskStatusRunning && task.StartedAt == nil {
			task.StartedAt = &now
		}
		if req.Status == common.TaskStatusCompleted {
			task.CompletedAt = &now
		}
	}
	m.tasksMutex.Unlock()

	// Actualizar progreso del job
	if jobID != "" {
		go m.updateJobProgress(jobID)
	}

	// Si una tarea se completó, verificar si hay tareas dependientes listas para asignar
	if req.Status == common.TaskStatusCompleted && jobID != "" {
		go m.checkAndAssignReadyTasks(jobID)
	}

	// Si una tarea falló, intentar reasignar o marcar job como fallido
	if req.Status == common.TaskStatusFailed && jobID != "" {
		go m.handleTaskFailure(jobID, req.TaskID)
	}

	// Responder
	resp := common.TaskUpdateResponse{
		Success: true,
		Message: "Tarea actualizada",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// updateJobProgress actualiza el progreso de un job basado en sus tareas
func (m *Master) updateJobProgress(jobID string) {
	m.jobsMutex.Lock()
	defer m.jobsMutex.Unlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return
	}

	var completed, failed, running, total int
	for _, task := range job.Tasks {
		total++
		switch task.Status {
		case common.TaskStatusCompleted:
			completed++
		case common.TaskStatusFailed:
			failed++
		case common.TaskStatusRunning:
			running++
		}
	}

	// Actualizar progreso
	if total > 0 {
		job.Progress = (float64(completed) / float64(total)) * 100.0
	}

	// Actualizar estado del job
	if completed == total {
		job.Status = common.JobStatusSucceeded
		job.CompletedAt = time.Now()
		m.logger.Info("✓ Job %s completado exitosamente (%d tareas)", jobID, total)
	} else if failed > 0 && (completed+failed) == total {
		job.Status = common.JobStatusFailed
		job.CompletedAt = time.Now()
		m.logger.Error("✗ Job %s fallido (%d/%d tareas fallidas)", jobID, failed, total)
	} else if running > 0 {
		job.Status = common.JobStatusRunning
	}

	job.UpdatedAt = time.Now()
}

// handleTaskFailure maneja el fallo de una tarea
func (m *Master) handleTaskFailure(jobID, taskID string) {
	m.tasksMutex.Lock()
	task, exists := m.tasks[taskID]
	m.tasksMutex.Unlock()

	if !exists {
		return
	}

	task.AttemptNum++

	if task.AttemptNum >= m.config.MaxRetries {
		m.logger.Error("Tarea %s alcanzó máximo de reintentos (%d)", taskID, m.config.MaxRetries)
		// Ya está marcada como FAILED, actualizar progreso del job
		go m.updateJobProgress(jobID)
	} else {
		// Intentar reasignar
		m.logger.Info("Reintentando tarea %s (intento %d/%d)", taskID, task.AttemptNum, m.config.MaxRetries)
		if err := m.scheduler.ReassignFailedTask(task); err != nil {
			m.logger.Error("Error reasignando tarea %s: %v", taskID, err)
			task.Status = common.TaskStatusFailed
			go m.updateJobProgress(jobID)
		}
	}
}

// checkAndAssignReadyTasks verifica y asigna tareas cuyas dependencias están completas
func (m *Master) checkAndAssignReadyTasks(jobID string) {
	m.jobsMutex.RLock()
	job, exists := m.jobs[jobID]
	m.jobsMutex.RUnlock()

	if !exists {
		return
	}

	m.tasksMutex.Lock()
	defer m.tasksMutex.Unlock()

	m.workersMutex.RLock()
	defer m.workersMutex.RUnlock()

	// Obtener workers disponibles
	var availableWorkers []*common.WorkerInfo
	for _, worker := range m.workers {
		if worker.Status == common.WorkerUp {
			availableWorkers = append(availableWorkers, worker)
		}
	}

	if len(availableWorkers) == 0 {
		return
	}

	tasksAssigned := 0

	// Revisar todas las tareas pendientes del job
	for _, task := range job.Tasks {
		if task.Status != common.TaskPending {
			continue
		}

		// Verificar si todas las dependencias están completas
		allDepsComplete := true
		for _, depNodeID := range task.Dependencies {
			// Buscar tareas del nodo dependencia
			depComplete := false
			for _, depTask := range job.Tasks {
				if depTask.NodeID == depNodeID && depTask.Status == common.TaskStatusCompleted {
					depComplete = true
					break
				}
			}
			if !depComplete {
				allDepsComplete = false
				break
			}
		}

		// Si todas las dependencias están completas, asignar la tarea
		if allDepsComplete {
			// Balanceo por carga: elegir worker con menos tareas activas
			minLoad := int(^uint(0) >> 1)
			var selectedWorker *common.WorkerInfo

			for _, worker := range availableWorkers {
				if worker.ActiveTasks < minLoad {
					minLoad = worker.ActiveTasks
					selectedWorker = worker
				}
			}

			if selectedWorker != nil {
				task.WorkerID = selectedWorker.ID
				task.Status = common.TaskAssigned
				m.tasks[task.ID] = task
				selectedWorker.ActiveTasks++
				tasksAssigned++
			}
		}
	}

	if tasksAssigned > 0 {
		m.logger.Info("Asignadas %d tareas del job %s después de completitud de dependencias", tasksAssigned, jobID)
	}
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
			var failedWorkers []string
			for _, worker := range m.workers {
				if now.Sub(worker.LastHeartbeat) > timeoutDuration {
					if worker.Status == common.WorkerStatusUp {
						m.logger.Warn("Worker %s no responde (último heartbeat: %v)", worker.ID, worker.LastHeartbeat)
						worker.Status = common.WorkerStatusDown
						failedWorkers = append(failedWorkers, worker.ID)
					}
				}
			}
			m.workersMutex.Unlock()

			// Reasignar tareas de workers caídos
			if len(failedWorkers) > 0 {
				go m.handleWorkerFailures(failedWorkers)
			}
		case <-m.shutdownChan:
			m.logger.Info("Deteniendo monitor de heartbeats")
			return
		}
	}
}

// handleWorkerFailures maneja fallos de workers y reasigna sus tareas
func (m *Master) handleWorkerFailures(failedWorkers []string) {
	m.tasksMutex.Lock()
	defer m.tasksMutex.Unlock()

	for _, workerID := range failedWorkers {
		m.logger.Info("Procesando fallo del worker %s", workerID)

		// Buscar todas las tareas asignadas o en ejecución en este worker
		for _, task := range m.tasks {
			if task.WorkerID == workerID && (task.Status == common.TaskAssigned || task.Status == common.TaskRunning) {
				m.logger.Warn("Tarea %s estaba en worker caído %s (intento %d)", task.ID, workerID, task.AttemptNum)

				// Incrementar contador de intentos
				task.AttemptNum++

				if task.AttemptNum >= m.config.MaxRetries {
					m.logger.Error("Tarea %s alcanzó máximo de reintentos (%d)", task.ID, m.config.MaxRetries)
					task.Status = common.TaskStatusFailed
					task.Error = fmt.Sprintf("Worker failed and max retries (%d) exceeded", m.config.MaxRetries)

					// Marcar job como fallido si todas las tareas fallaron
					go m.checkJobFailure(task.JobID)
				} else {
					// Reasignar tarea
					m.logger.Info("Reasignando tarea %s (intento %d/%d)", task.ID, task.AttemptNum, m.config.MaxRetries)
					if err := m.scheduler.ReassignFailedTask(task); err != nil {
						m.logger.Error("Error reasignando tarea %s: %v", task.ID, err)
						task.Status = common.TaskStatusFailed
						task.Error = fmt.Sprintf("Failed to reassign: %v", err)
					}
				}
			}
		}
	}
}

// checkJobFailure verifica si un job debe marcarse como fallido
func (m *Master) checkJobFailure(jobID string) {
	m.jobsMutex.Lock()
	defer m.jobsMutex.Unlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return
	}

	// Contar tareas por estado
	var completed, failed, total int
	for _, task := range job.Tasks {
		total++
		if task.Status == common.TaskStatusCompleted {
			completed++
		} else if task.Status == common.TaskStatusFailed {
			failed++
		}
	}

	// Si todas las tareas fallaron o no se pueden completar
	if failed > 0 && (completed+failed) == total {
		job.Status = common.JobStatusFailed
		job.CompletedAt = time.Now()
		job.UpdatedAt = time.Now()
		m.logger.Error("Job %s marcado como FAILED (%d/%d tareas fallidas)", jobID, failed, total)
	} else if completed == total {
		job.Status = common.JobStatusSucceeded
		job.CompletedAt = time.Now()
		job.UpdatedAt = time.Now()
		job.Progress = 100.0
		m.logger.Info("Job %s completado exitosamente (%d tareas)", jobID, total)
	} else {
		// Actualizar progreso
		job.Progress = (float64(completed) / float64(total)) * 100.0
		job.UpdatedAt = time.Now()
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

	// Canal para señales de sistema
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Iniciar master en goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := master.Start(); err != nil {
			errChan <- err
		}
	}()

	// Esperar señal de terminación o error
	select {
	case <-sigChan:
		master.logger.Info("Señal de terminación recibida")
		master.Shutdown()
		master.logger.Info("Master detenido correctamente")
	case err := <-errChan:
		master.logger.Error("Error iniciando Master: %v", err)
	}
}
