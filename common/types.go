package common

import "time"

// JobStatus representa el estado actual de un job en el sistema
type JobStatus string

const (
	JobStatusAccepted  JobStatus = "ACCEPTED"
	JobStatusRunning   JobStatus = "RUNNING"
	JobStatusFailed    JobStatus = "FAILED"
	JobStatusSucceeded JobStatus = "SUCCEEDED"
)

// TaskStatus representa el estado de una tarea individual
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "PENDING"
	TaskStatusRunning   TaskStatus = "RUNNING"
	TaskStatusCompleted TaskStatus = "COMPLETED"
	TaskStatusFailed    TaskStatus = "FAILED"
)

// OperatorType define los tipos de operadores soportados
type OperatorType string

const (
	OpReadCSV      OperatorType = "read_csv"
	OpMap          OperatorType = "map"
	OpFilter       OperatorType = "filter"
	OpFlatMap      OperatorType = "flat_map"
	OpReduceByKey  OperatorType = "reduce_by_key"
	OpJoin         OperatorType = "join"
)

// WorkerStatus representa el estado de un worker
type WorkerStatus string

const (
	WorkerStatusUp   WorkerStatus = "UP"
	WorkerStatusDown WorkerStatus = "DOWN"
)

// DAGNode representa un nodo (operador) en el grafo de ejecución
type DAGNode struct {
	ID         string                 `json:"id"`
	Operator   OperatorType           `json:"op"`
	Path       string                 `json:"path,omitempty"`
	Function   string                 `json:"fn,omitempty"`
	Key        string                 `json:"key,omitempty"`
	Partitions int                    `json:"partitions,omitempty"`
	Params     map[string]interface{} `json:"params,omitempty"`
}

// DAGEdge representa una conexión entre dos nodos del DAG
type DAGEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// DAG representa el grafo acíclico dirigido de un job
type DAG struct {
	Nodes []DAGNode `json:"nodes"`
	Edges []DAGEdge `json:"edges"`
}

// Job representa un trabajo de procesamiento batch
type Job struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	DAG         DAG       `json:"dag"`
	Parallelism int       `json:"parallelism"`
	Status      JobStatus `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Progress    float64   `json:"progress"`
}

// Task representa una unidad de ejecución asignada a un worker
type Task struct {
	ID          string                 `json:"id"`
	JobID       string                 `json:"job_id"`
	NodeID      string                 `json:"node_id"`
	Operator    OperatorType           `json:"operator"`
	Partition   int                    `json:"partition"`
	AttemptNum  int                    `json:"attempt_num"`
	Status      TaskStatus             `json:"status"`
	WorkerID    string                 `json:"worker_id,omitempty"`
	Input       []string               `json:"input,omitempty"`
	Output      string                 `json:"output,omitempty"`
	Params      map[string]interface{} `json:"params,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// WorkerInfo contiene información sobre un worker registrado
type WorkerInfo struct {
	ID            string       `json:"id"`
	Address       string       `json:"address"`
	Status        WorkerStatus `json:"status"`
	RegisteredAt  time.Time    `json:"registered_at"`
	LastHeartbeat time.Time    `json:"last_heartbeat"`
	ActiveTasks   int          `json:"active_tasks"`
	TotalTasks    int          `json:"total_tasks"`
}

// JobMetrics contiene métricas de ejecución de un job
type JobMetrics struct {
	JobID          string        `json:"job_id"`
	TotalTasks     int           `json:"total_tasks"`
	CompletedTasks int           `json:"completed_tasks"`
	FailedTasks    int           `json:"failed_tasks"`
	Duration       time.Duration `json:"duration"`
	Throughput     float64       `json:"throughput"`
}
