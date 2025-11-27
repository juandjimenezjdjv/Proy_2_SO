package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/juandjimenezjdjv/Proy_2_SO/common"
)

// Worker es el nodo de ejecución que procesa tareas asignadas por el master
type Worker struct {
	id            string
	address       string
	masterAddress string
	config        *common.Config
	logger        *common.Logger

	// Estado interno
	activeTasks map[string]*common.Task
	tasksMutex  sync.RWMutex

	// Métricas
	metricsCollector *common.MetricsCollector

	// Control
	heartbeatTicker *time.Ticker
	shutdownChan    chan bool
	registered      bool
}

// NewWorker crea una nueva instancia del Worker
func NewWorker(config *common.Config) *Worker {
	hostname, _ := os.Hostname()
	workerID := fmt.Sprintf("worker-%s-%d", hostname, time.Now().Unix())

	// Determinar dirección del worker (puerto aleatorio por ahora)
	address := fmt.Sprintf("worker-%d:8081", time.Now().Unix()%1000)

	return &Worker{
		id:               workerID,
		address:          address,
		masterAddress:    fmt.Sprintf("http://%s:%d", config.MasterHost, config.MasterPort),
		config:           config,
		logger:           common.NewLogger("WORKER", config.LogLevel),
		activeTasks:      make(map[string]*common.Task),
		metricsCollector: common.NewMetricsCollector(),
		shutdownChan:     make(chan bool),
		registered:       false,
	}
}

// Start inicia el worker y se registra con el master
func (w *Worker) Start() error {
	w.logger.Info("Iniciando Worker %s", w.id)

	// Registrarse con el master
	if err := w.registerWithMaster(); err != nil {
		return fmt.Errorf("error registrando con master: %v", err)
	}

	// Iniciar envío de heartbeats
	go w.startHeartbeat()

	// Iniciar polling de tareas
	go w.startTaskPolling()

	// Esperar señal de shutdown
	w.waitForShutdown()

	return nil
}

// registerWithMaster registra el worker con el master
func (w *Worker) registerWithMaster() error {
	w.logger.Info("Registrando con Master en %s", w.masterAddress)

	req := common.RegisterWorkerRequest{
		WorkerID: w.id,
		Address:  w.address,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error serializando registro: %v", err)
	}

	url := fmt.Sprintf("%s/api/v1/workers/register", w.masterAddress)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("error enviando registro: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("master rechazó registro: status %d", resp.StatusCode)
	}

	var regResp common.RegisterWorkerResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return fmt.Errorf("error decodificando respuesta: %v", err)
	}

	if !regResp.Success {
		return fmt.Errorf("registro fallido: %s", regResp.Message)
	}

	w.registered = true
	w.logger.Info("Registrado exitosamente con Master %s (heartbeat cada %d segundos)",
		regResp.MasterID, regResp.HeartbeatSec)

	return nil
}

// startHeartbeat inicia el envío periódico de heartbeats al master
func (w *Worker) startHeartbeat() {
	w.heartbeatTicker = time.NewTicker(time.Duration(w.config.HeartbeatSec) * time.Second)
	defer w.heartbeatTicker.Stop()

	for {
		select {
		case <-w.heartbeatTicker.C:
			if err := w.sendHeartbeat(); err != nil {
				w.logger.Error("Error enviando heartbeat: %v", err)
			}
		case <-w.shutdownChan:
			w.logger.Info("Deteniendo heartbeats")
			return
		}
	}
}

// sendHeartbeat envía un heartbeat al master
func (w *Worker) sendHeartbeat() error {
	w.tasksMutex.RLock()
	activeTasks := len(w.activeTasks)
	w.tasksMutex.RUnlock()

	// Recolectar métricas del sistema
	metrics := w.metricsCollector.Collect()

	req := common.HeartbeatRequest{
		WorkerID:    w.id,
		ActiveTasks: activeTasks,
		Metrics:     metrics,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error serializando heartbeat: %v", err)
	}

	url := fmt.Sprintf("%s/api/v1/workers/heartbeat", w.masterAddress)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("error enviando heartbeat: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("master rechazó heartbeat: status %d", resp.StatusCode)
	}

	w.logger.Debug("Heartbeat enviado (tareas activas: %d, %s)", activeTasks, metrics.FormatMetrics())
	return nil
}

// startTaskPolling inicia el polling de tareas desde el master
func (w *Worker) startTaskPolling() {
	// Esperar un poco para que el heartbeat se establezca
	time.Sleep(2 * time.Second)

	ticker := time.NewTicker(5 * time.Second) // Poll cada 5 segundos
	defer ticker.Stop()

	w.logger.Info("Iniciando polling de tareas...")

	for {
		select {
		case <-ticker.C:
			if err := w.pollAndExecuteTasks(); err != nil {
				w.logger.Error("Error en polling de tareas: %v", err)
			}
		case <-w.shutdownChan:
			w.logger.Info("Deteniendo polling de tareas")
			return
		}
	}
}

// pollAndExecuteTasks obtiene y ejecuta tareas asignadas desde el master
func (w *Worker) pollAndExecuteTasks() error {
	// Obtener tareas asignadas
	url := fmt.Sprintf("%s/api/v1/workers/tasks?worker_id=%s", w.masterAddress, w.id)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error obteniendo tareas: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("master devolvió status %d", resp.StatusCode)
	}

	var tasks []*common.Task
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return fmt.Errorf("error decodificando tareas: %v", err)
	}

	// Ejecutar cada tarea en una goroutine
	for _, task := range tasks {
		// Verificar si ya está siendo ejecutada
		w.tasksMutex.RLock()
		_, exists := w.activeTasks[task.ID]
		w.tasksMutex.RUnlock()

		if exists {
			continue // Ya está en ejecución
		}

		w.logger.Info("Nueva tarea recibida: %s (operador: %s)", task.ID, task.Operator)

		// Marcar como activa
		w.tasksMutex.Lock()
		w.activeTasks[task.ID] = task
		w.tasksMutex.Unlock()

		// Ejecutar en goroutine
		go func(t *common.Task) {
			if err := w.executeTask(t); err != nil {
				w.logger.Error("Error ejecutando tarea %s: %v", t.ID, err)
			}

			// Remover de tareas activas
			w.tasksMutex.Lock()
			delete(w.activeTasks, t.ID)
			w.tasksMutex.Unlock()
		}(task)
	}

	return nil
}

// executeTask ejecuta una tarea asignada usando el executor
func (w *Worker) executeTask(task *common.Task) error {
	startTime := time.Now()
	w.logger.Info("Ejecutando tarea %s (operador: %s)", task.ID, task.Operator)

	// Reportar inicio
	if err := w.reportTaskStatus(task.ID, common.TaskStatusRunning, "", 0); err != nil {
		w.logger.Error("Error reportando inicio de tarea: %v", err)
	}

	// Ejecutar con límites de tiempo y memoria
	err := w.executeTaskWithLimits(task)
	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		w.logger.Error("Error en ejecución de tarea %s: %v (duración: %dms)", task.ID, err, duration)
		// Reportar falla
		w.reportTaskStatus(task.ID, common.TaskStatusFailed, err.Error(), duration)
		return err
	}

	// Reportar completitud
	if err := w.reportTaskStatus(task.ID, common.TaskStatusCompleted, "", duration); err != nil {
		w.logger.Error("Error reportando tarea completada: %v", err)
		return err
	}

	w.logger.Info("Tarea %s completada exitosamente (duración: %dms)", task.ID, duration)
	return nil
}

// executeTaskWithLimits ejecuta una tarea con límites de tiempo y memoria
func (w *Worker) executeTaskWithLimits(task *common.Task) error {
	// Crear contexto con timeout si está configurado
	ctx := context.Background()
	var cancel context.CancelFunc

	if task.TimeoutSec > 0 {
		w.logger.Info("Tarea %s: timeout configurado a %d segundos", task.ID, task.TimeoutSec)
		ctx, cancel = context.WithTimeout(ctx, time.Duration(task.TimeoutSec)*time.Second)
		defer cancel()
	} else {
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
	}

	// Canal para recibir el resultado de la ejecución
	type result struct {
		err error
	}
	resultChan := make(chan result, 1)

	// Ejecutar tarea en goroutine
	go func() {
		err := ExecuteTask(task)
		resultChan <- result{err: err}
	}()

	// Monitoreo de memoria si está configurado
	var memTicker *time.Ticker
	if task.MaxMemoryMB > 0 {
		w.logger.Info("Tarea %s: límite de memoria configurado a %d MB", task.ID, task.MaxMemoryMB)
		memTicker = time.NewTicker(500 * time.Millisecond)
		defer memTicker.Stop()
	}

	// Esperar resultado o límites excedidos
	for {
		select {
		case <-ctx.Done():
			// Timeout excedido
			if ctx.Err() == context.DeadlineExceeded {
				w.logger.Error("Tarea %s: timeout de %d segundos excedido", task.ID, task.TimeoutSec)
				return fmt.Errorf("timeout de %d segundos excedido", task.TimeoutSec)
			}
			return ctx.Err()

		case res := <-resultChan:
			// Tarea completada (éxito o error)
			return res.err

		case <-func() <-chan time.Time {
			if memTicker != nil {
				return memTicker.C
			}
			return nil
		}():
			// Verificar uso de memoria
			if task.MaxMemoryMB > 0 {
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				currentMemMB := int64(m.Alloc / 1024 / 1024)

				if currentMemMB > task.MaxMemoryMB {
					w.logger.Error("Tarea %s: límite de memoria excedido (%d MB > %d MB)", task.ID, currentMemMB, task.MaxMemoryMB)
					cancel() // Cancelar contexto
					return fmt.Errorf("límite de memoria de %d MB excedido (uso actual: %d MB)", task.MaxMemoryMB, currentMemMB)
				}
			}
		}
	}
}

// reportTaskStatus reporta el estado de una tarea al master
func (w *Worker) reportTaskStatus(taskID string, status common.TaskStatus, errorMsg string, durationMs int64) error {
	req := common.TaskUpdateRequest{
		TaskID:     taskID,
		Status:     status,
		Progress:   100.0,
		Error:      errorMsg,
		DurationMs: durationMs,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error serializando actualización: %v", err)
	}

	url := fmt.Sprintf("%s/api/v1/tasks/update", w.masterAddress)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("error enviando actualización: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("master rechazó actualización: status %d", resp.StatusCode)
	}

	return nil
}

// waitForShutdown espera una señal de shutdown y limpia recursos
func (w *Worker) waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	w.logger.Info("Señal de shutdown recibida")
	w.Shutdown()
}

// Shutdown detiene el worker de forma ordenada
func (w *Worker) Shutdown() {
	w.logger.Info("Iniciando apagado del Worker...")
	close(w.shutdownChan)

	// Esperar un momento para que los heartbeats se detengan
	time.Sleep(1 * time.Second)

	w.logger.Info("Worker apagado completamente")
}

func main() {
	config := common.LoadConfig()
	worker := NewWorker(config)

	if err := worker.Start(); err != nil {
		worker.logger.Error("Error iniciando Worker: %v", err)
		os.Exit(1)
	}
}
