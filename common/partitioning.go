package common

import (
	"fmt"
	"hash/fnv"
)

// PartitionStrategy define la estrategia de particionamiento
type PartitionStrategy string

const (
	// PartitionHash usa hash de la clave para distribuir datos
	PartitionHash PartitionStrategy = "hash"
	// PartitionRoundRobin distribuye datos secuencialmente
	PartitionRoundRobin PartitionStrategy = "round-robin"
	// PartitionRange particiona por rangos de valores
	PartitionRange PartitionStrategy = "range"
)

// Partitioner gestiona la distribución de datos en particiones
type Partitioner struct {
	strategy      PartitionStrategy
	numPartitions int
}

// NewPartitioner crea un nuevo particionador
func NewPartitioner(strategy PartitionStrategy, numPartitions int) *Partitioner {
	if numPartitions <= 0 {
		numPartitions = 1
	}
	return &Partitioner{
		strategy:      strategy,
		numPartitions: numPartitions,
	}
}

// GetPartition determina a qué partición pertenece un registro
func (p *Partitioner) GetPartition(key string) int {
	switch p.strategy {
	case PartitionHash:
		return p.hashPartition(key)
	case PartitionRoundRobin:
		// Round-robin requiere mantener estado, por defecto usar hash
		return p.hashPartition(key)
	case PartitionRange:
		return p.rangePartition(key)
	default:
		return p.hashPartition(key)
	}
}

// hashPartition usa FNV-1a hash para distribuir uniformemente
func (p *Partitioner) hashPartition(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32()) % p.numPartitions
}

// rangePartition particiona por rangos alfabéticos (simple)
func (p *Partitioner) rangePartition(key string) int {
	if len(key) == 0 {
		return 0
	}
	// Distribuir por primera letra
	firstChar := int(key[0])
	return (firstChar * p.numPartitions) / 256
}

// BalancePartitions calcula distribución óptima de particiones entre workers
func BalancePartitions(totalPartitions, numWorkers int) [][]int {
	if numWorkers <= 0 {
		return nil
	}

	// Asignar particiones de forma balanceada
	assignment := make([][]int, numWorkers)
	for i := 0; i < totalPartitions; i++ {
		workerIdx := i % numWorkers
		assignment[workerIdx] = append(assignment[workerIdx], i)
	}

	return assignment
}

// ShuffleKey genera clave para shuffle en operaciones como join
func ShuffleKey(key string, partitionID int) string {
	return fmt.Sprintf("%s#%d", key, partitionID)
}

// ExtractKey extrae la clave original de una shuffle key
func ExtractKey(shuffleKey string) string {
	// Simple: retornar todo antes del #
	for i := len(shuffleKey) - 1; i >= 0; i-- {
		if shuffleKey[i] == '#' {
			return shuffleKey[:i]
		}
	}
	return shuffleKey
}

// PartitionInfo contiene información sobre una partición
type PartitionInfo struct {
	ID          int    // Número de partición
	WorkerID    string // Worker asignado
	RecordCount int    // Número de registros
	SizeBytes   int64  // Tamaño en bytes
	Status      string // Estado: pending, processing, completed, failed
}

// CoalescePartitions combina múltiples particiones pequeñas
// Útil después de filtros que reducen mucho los datos
func CoalescePartitions(currentPartitions, targetPartitions int) map[int][]int {
	if targetPartitions >= currentPartitions {
		// No hay que combinar
		mapping := make(map[int][]int)
		for i := 0; i < currentPartitions; i++ {
			mapping[i] = []int{i}
		}
		return mapping
	}

	// Mapear particiones antiguas a nuevas
	mapping := make(map[int][]int)
	for i := 0; i < currentPartitions; i++ {
		newPartition := i % targetPartitions
		mapping[newPartition] = append(mapping[newPartition], i)
	}

	return mapping
}

// RepartitionStrategy decide si reparticionar datos
type RepartitionStrategy struct {
	// MinRecordsPerPartition: si una partición tiene menos registros, coalescer
	MinRecordsPerPartition int
	// MaxRecordsPerPartition: si una partición tiene más registros, dividir
	MaxRecordsPerPartition int
	// TargetPartitions: número ideal de particiones
	TargetPartitions int
}

// ShouldRepartition determina si es necesario reparticionar
func (rs *RepartitionStrategy) ShouldRepartition(partitions []PartitionInfo) bool {
	if len(partitions) == 0 {
		return false
	}

	// Verificar si hay particiones muy desbalanceadas
	totalRecords := 0
	for _, p := range partitions {
		totalRecords += p.RecordCount
	}

	avgRecords := totalRecords / len(partitions)

	// Si hay desbalance mayor al 50%, reparticionar
	for _, p := range partitions {
		if p.RecordCount < avgRecords/2 || p.RecordCount > avgRecords*2 {
			return true
		}
	}

	return false
}
