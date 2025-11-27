package common

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheManager maneja el cache en memoria con spill automático a disco
type CacheManager struct {
	maxMemoryMB     int64
	currentMemoryMB int64
	spillDir        string
	cache           map[string][][]string
	spilledFiles    map[string]string
	mu              sync.RWMutex
	logger          *Logger
}

// NewCacheManager crea un nuevo gestor de cache
func NewCacheManager(maxMemoryMB int64, spillDir string, logger *Logger) *CacheManager {
	if maxMemoryMB <= 0 {
		maxMemoryMB = 100 // Default: 100MB
	}

	// Crear directorio de spill si no existe
	os.MkdirAll(spillDir, 0755)

	return &CacheManager{
		maxMemoryMB:  maxMemoryMB,
		spillDir:     spillDir,
		cache:        make(map[string][][]string),
		spilledFiles: make(map[string]string),
		logger:       logger,
	}
}

// Put almacena datos en cache, hace spill a disco si excede límite
func (cm *CacheManager) Put(key string, data [][]string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Estimar tamaño de los datos (aproximado)
	estimatedMB := cm.estimateSize(data)

	// Si agregar estos datos excede el límite, hacer spill
	if cm.currentMemoryMB+estimatedMB > cm.maxMemoryMB {
		cm.logger.Info(fmt.Sprintf("Cache excede límite (%dMB/%dMB), haciendo spill a disco para key: %s",
			cm.currentMemoryMB+estimatedMB, cm.maxMemoryMB, key))

		if err := cm.spillToDisk(key, data); err != nil {
			return fmt.Errorf("error en spill: %w", err)
		}
		return nil
	}

	// Almacenar en memoria
	cm.cache[key] = data
	cm.currentMemoryMB += estimatedMB
	cm.logger.Debug(fmt.Sprintf("Datos almacenados en cache: %s (%dMB, total: %dMB/%dMB)",
		key, estimatedMB, cm.currentMemoryMB, cm.maxMemoryMB))

	return nil
}

// Get recupera datos del cache o disco
func (cm *CacheManager) Get(key string) ([][]string, error) {
	cm.mu.RLock()

	// Primero buscar en memoria
	if data, ok := cm.cache[key]; ok {
		cm.mu.RUnlock()
		cm.logger.Debug(fmt.Sprintf("Cache HIT en memoria: %s", key))
		return data, nil
	}

	// Buscar en archivos spilled
	spillPath, ok := cm.spilledFiles[key]
	cm.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("key no encontrada en cache: %s", key)
	}

	// Leer desde disco
	cm.logger.Debug(fmt.Sprintf("Cache MISS, leyendo desde disco: %s", spillPath))
	return cm.readFromDisk(spillPath)
}

// spillToDisk escribe datos a disco y libera memoria
func (cm *CacheManager) spillToDisk(key string, data [][]string) error {
	// Generar nombre de archivo único
	spillPath := filepath.Join(cm.spillDir, fmt.Sprintf("spill-%s-%d.csv", key, time.Now().UnixNano()))

	// Escribir a disco
	file, err := os.Create(spillPath)
	if err != nil {
		return fmt.Errorf("error creando archivo spill: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, record := range data {
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("error escribiendo registro: %w", err)
		}
	}

	// Registrar archivo spilled
	cm.spilledFiles[key] = spillPath

	// Si estaba en memoria, remover
	if _, ok := cm.cache[key]; ok {
		estimatedMB := cm.estimateSize(cm.cache[key])
		delete(cm.cache, key)
		cm.currentMemoryMB -= estimatedMB
	}

	cm.logger.Info(fmt.Sprintf("Spill completado: %s -> %s", key, spillPath))
	return nil
}

// readFromDisk lee datos desde un archivo spilled
func (cm *CacheManager) readFromDisk(path string) ([][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error abriendo archivo spill: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error leyendo archivo spill: %w", err)
	}

	return records, nil
}

// estimateSize estima el tamaño en MB de los datos (aproximado)
func (cm *CacheManager) estimateSize(data [][]string) int64 {
	if len(data) == 0 {
		return 0
	}

	// Estimar: número de registros * columnas promedio * bytes por string
	totalChars := 0
	for _, record := range data {
		for _, field := range record {
			totalChars += len(field)
		}
	}

	// Agregar overhead de estructuras
	totalBytes := totalChars + (len(data) * 100) // 100 bytes overhead por registro
	totalMB := int64(totalBytes / (1024 * 1024))

	if totalMB < 1 {
		totalMB = 1 // Mínimo 1MB
	}

	return totalMB
}

// Clear limpia todo el cache y archivos spilled
func (cm *CacheManager) Clear() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Limpiar memoria
	cm.cache = make(map[string][][]string)
	cm.currentMemoryMB = 0

	// Eliminar archivos spilled
	for key, path := range cm.spilledFiles {
		if err := os.Remove(path); err != nil {
			cm.logger.Warn(fmt.Sprintf("Error eliminando archivo spill %s: %v", path, err))
		}
		delete(cm.spilledFiles, key)
	}

	cm.logger.Info("Cache limpiado completamente")
	return nil
}

// GetStats retorna estadísticas del cache
func (cm *CacheManager) GetStats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return map[string]interface{}{
		"max_memory_mb":     cm.maxMemoryMB,
		"current_memory_mb": cm.currentMemoryMB,
		"cached_keys":       len(cm.cache),
		"spilled_keys":      len(cm.spilledFiles),
		"memory_usage_pct":  float64(cm.currentMemoryMB) / float64(cm.maxMemoryMB) * 100,
	}
}
