package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
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
		id:            workerID,
		address:       address,
		masterAddress: fmt.Sprintf("http://%s:%d", config.MasterHost, config.MasterPort),
		config:        config,
		logger:        common.NewLogger("WORKER", config.LogLevel),
		activeTasks:   make(map[string]*common.Task),
		shutdownChan:  make(chan bool),
		registered:    false,
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

	req := common.HeartbeatRequest{
		WorkerID:    w.id,
		ActiveTasks: activeTasks,
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

	w.logger.Debug("Heartbeat enviado (tareas activas: %d)", activeTasks)
	return nil
}

// executeTask ejecuta una tarea asignada (placeholder por ahora)
func (w *Worker) executeTask(task *common.Task) error {
	w.logger.Info("Ejecutando tarea %s (operador: %s)", task.ID, task.Operator)

	// Agregar a tareas activas
	w.tasksMutex.Lock()
	w.activeTasks[task.ID] = task
	w.tasksMutex.Unlock()

	// Simular ejecución (placeholder)
	time.Sleep(2 * time.Second)

	// Remover de tareas activas
	w.tasksMutex.Lock()
	delete(w.activeTasks, task.ID)
	w.tasksMutex.Unlock()

	// Reportar completitud
	if err := w.reportTaskStatus(task.ID, common.TaskStatusCompleted, ""); err != nil {
		w.logger.Error("Error reportando tarea completada: %v", err)
		return err
	}

	w.logger.Info("Tarea %s completada exitosamente", task.ID)
	return nil
}

// reportTaskStatus reporta el estado de una tarea al master
func (w *Worker) reportTaskStatus(taskID string, status common.TaskStatus, errorMsg string) error {
	req := common.TaskUpdateRequest{
		TaskID:   taskID,
		Status:   status,
		Progress: 100.0,
		Error:    errorMsg,
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
