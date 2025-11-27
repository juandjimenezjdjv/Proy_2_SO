package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// StateManager maneja la persistencia del estado del master
type StateManager struct {
	storageDir string
	logger     *Logger
	mu         sync.RWMutex
	autoSave   bool
	saveTimer  *time.Timer
}

// StateSnapshot representa un snapshot del estado del sistema
type StateSnapshot struct {
	Timestamp time.Time              `json:"timestamp"`
	Jobs      map[string]*Job        `json:"jobs"`
	Tasks     map[string]*Task       `json:"tasks"`
	Workers   map[string]*WorkerInfo `json:"workers"`
}

// NewStateManager crea un nuevo gestor de estado
func NewStateManager(storageDir string, logger *Logger, autoSave bool) *StateManager {
	// Crear directorio si no existe
	os.MkdirAll(storageDir, 0755)

	return &StateManager{
		storageDir: storageDir,
		logger:     logger,
		autoSave:   autoSave,
	}
}

// SaveState guarda el estado actual a disco
func (sm *StateManager) SaveState(jobs map[string]*Job, tasks map[string]*Task, workers map[string]*WorkerInfo) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	snapshot := StateSnapshot{
		Timestamp: time.Now(),
		Jobs:      jobs,
		Tasks:     tasks,
		Workers:   workers,
	}

	// Serializar a JSON
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializando estado: %w", err)
	}

	// Guardar con timestamp
	filename := fmt.Sprintf("state-%d.json", time.Now().Unix())
	filePath := filepath.Join(sm.storageDir, filename)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("error escribiendo archivo: %w", err)
	}

	// También guardar como "latest"
	latestPath := filepath.Join(sm.storageDir, "state-latest.json")
	if err := os.WriteFile(latestPath, data, 0644); err != nil {
		sm.logger.Warn("Error guardando latest state: %v", err)
	}

	sm.logger.Info("Estado guardado: %s (%d jobs, %d tasks, %d workers)",
		filename, len(jobs), len(tasks), len(workers))

	return nil
}

// LoadLatestState carga el último estado guardado
func (sm *StateManager) LoadLatestState() (*StateSnapshot, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	latestPath := filepath.Join(sm.storageDir, "state-latest.json")

	// Verificar si existe
	if _, err := os.Stat(latestPath); os.IsNotExist(err) {
		sm.logger.Info("No se encontró estado previo")
		return nil, nil
	}

	// Leer archivo
	data, err := os.ReadFile(latestPath)
	if err != nil {
		return nil, fmt.Errorf("error leyendo archivo: %w", err)
	}

	// Deserializar
	var snapshot StateSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("error deserializando estado: %w", err)
	}

	sm.logger.Info("Estado cargado desde: %s (%d jobs, %d tasks, %d workers)",
		latestPath, len(snapshot.Jobs), len(snapshot.Tasks), len(snapshot.Workers))

	return &snapshot, nil
}

// StartAutoSave inicia guardado automático periódico
func (sm *StateManager) StartAutoSave(interval time.Duration, saveFunc func() error) {
	if !sm.autoSave {
		return
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if err := saveFunc(); err != nil {
				sm.logger.Error("Error en auto-save: %v", err)
			}
		}
	}()

	sm.logger.Info("Auto-save iniciado (intervalo: %v)", interval)
}

// CleanOldStates limpia estados antiguos manteniendo solo los últimos N
func (sm *StateManager) CleanOldStates(keepLast int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Leer todos los archivos de estado
	files, err := filepath.Glob(filepath.Join(sm.storageDir, "state-*.json"))
	if err != nil {
		return fmt.Errorf("error listando archivos: %w", err)
	}

	// Excluir state-latest.json
	stateFiles := make([]string, 0)
	for _, file := range files {
		if filepath.Base(file) != "state-latest.json" {
			stateFiles = append(stateFiles, file)
		}
	}

	// Si hay más de keepLast, eliminar los más antiguos
	if len(stateFiles) > keepLast {
		// Ordenar por nombre (timestamp está en el nombre)
		// Los más antiguos estarán primero
		toDelete := len(stateFiles) - keepLast
		for i := 0; i < toDelete; i++ {
			if err := os.Remove(stateFiles[i]); err != nil {
				sm.logger.Warn("Error eliminando %s: %v", stateFiles[i], err)
			} else {
				sm.logger.Debug("Estado antiguo eliminado: %s", stateFiles[i])
			}
		}
	}

	return nil
}

// GetStateFiles retorna lista de archivos de estado disponibles
func (sm *StateManager) GetStateFiles() ([]string, error) {
	files, err := filepath.Glob(filepath.Join(sm.storageDir, "state-*.json"))
	if err != nil {
		return nil, fmt.Errorf("error listando archivos: %w", err)
	}
	return files, nil
}
