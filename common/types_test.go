package common

import (
	"fmt"
	"testing"
	"time"
)

// TestJobStatus verifica los estados de job
func TestJobStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   JobStatus
		expected JobStatus
	}{
		{"Accepted status", JobStatusAccepted, "ACCEPTED"},
		{"Running status", JobStatusRunning, "RUNNING"},
		{"Failed status", JobStatusFailed, "FAILED"},
		{"Succeeded status", JobStatusSucceeded, "SUCCEEDED"},
		{"Pending alias", JobPending, "ACCEPTED"},
		{"Completed alias", JobCompleted, "SUCCEEDED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.status != tt.expected {
				t.Errorf("esperado %s, obtenido %s", tt.expected, tt.status)
			}
		})
	}
}

// TestTaskStatus verifica los estados de tarea
func TestTaskStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   TaskStatus
		expected TaskStatus
	}{
		{"Pending status", TaskStatusPending, "PENDING"},
		{"Running status", TaskStatusRunning, "RUNNING"},
		{"Completed status", TaskStatusCompleted, "COMPLETED"},
		{"Failed status", TaskStatusFailed, "FAILED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.status != tt.expected {
				t.Errorf("esperado %s, obtenido %s", tt.expected, tt.status)
			}
		})
	}
}

// TestOperatorType verifica los tipos de operadores
func TestOperatorType(t *testing.T) {
	tests := []struct {
		name     string
		operator OperatorType
		expected string
	}{
		{"ReadCSV", OpReadCSV, "read_csv"},
		{"Map", OpMap, "map"},
		{"Filter", OpFilter, "filter"},
		{"FlatMap", OpFlatMap, "flat_map"},
		{"ReduceByKey", OpReduceByKey, "reduce_by_key"},
		{"Aggregate", OpAggregate, "aggregate"},
		{"Join", OpJoin, "join"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.operator) != tt.expected {
				t.Errorf("esperado %s, obtenido %s", tt.expected, tt.operator)
			}
		})
	}
}

// TestWorkerStatus verifica los estados de worker
func TestWorkerStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   WorkerStatus
		expected WorkerStatus
	}{
		{"Up status", WorkerStatusUp, "UP"},
		{"Down status", WorkerStatusDown, "DOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.status != tt.expected {
				t.Errorf("esperado %s, obtenido %s", tt.expected, tt.status)
			}
		})
	}
}

// TestDAGNode verifica la creación de nodos DAG
func TestDAGNode(t *testing.T) {
	node := DAGNode{
		ID:         "node1",
		Operator:   OpMap,
		InputPaths: []string{"input.csv"},
		OutputPath: "output.csv",
		Partitions: 3,
		Params: map[string]interface{}{
			"function": "uppercase",
		},
	}

	if node.ID != "node1" {
		t.Errorf("esperado ID 'node1', obtenido '%s'", node.ID)
	}
	if node.Operator != OpMap {
		t.Errorf("esperado operador OpMap, obtenido %s", node.Operator)
	}
	if len(node.InputPaths) != 1 {
		t.Errorf("esperado 1 input path, obtenido %d", len(node.InputPaths))
	}
	if node.Partitions != 3 {
		t.Errorf("esperado 3 particiones, obtenido %d", node.Partitions)
	}
}

// TestDAG verifica la creación de DAG
func TestDAG(t *testing.T) {
	dag := DAG{
		Nodes: []DAGNode{
			{ID: "read", Operator: OpReadCSV},
			{ID: "map", Operator: OpMap},
			{ID: "reduce", Operator: OpReduceByKey},
		},
		Edges: []DAGEdge{
			{From: "read", To: "map"},
			{From: "map", To: "reduce"},
		},
	}

	if len(dag.Nodes) != 3 {
		t.Errorf("esperado 3 nodos, obtenido %d", len(dag.Nodes))
	}
	if len(dag.Edges) != 2 {
		t.Errorf("esperado 2 aristas, obtenido %d", len(dag.Edges))
	}

	// Verificar conectividad
	if dag.Edges[0].From != "read" || dag.Edges[0].To != "map" {
		t.Errorf("arista incorrecta: %v", dag.Edges[0])
	}
}

// TestJob verifica la creación y gestión de jobs
func TestJob(t *testing.T) {
	now := time.Now()
	job := Job{
		ID:          "job-001",
		Name:        "test-job",
		Status:      JobStatusAccepted,
		Parallelism: 3,
		SubmittedAt: now,
		Progress:    0.0,
	}

	if job.ID != "job-001" {
		t.Errorf("esperado ID 'job-001', obtenido '%s'", job.ID)
	}
	if job.Status != JobStatusAccepted {
		t.Errorf("esperado status ACCEPTED, obtenido %s", job.Status)
	}
	if job.Progress != 0.0 {
		t.Errorf("esperado progreso 0.0, obtenido %.2f", job.Progress)
	}

	// Simular progreso del job
	job.Status = JobStatusRunning
	job.StartedAt = time.Now()
	job.Progress = 50.0

	if job.Status != JobStatusRunning {
		t.Errorf("esperado status RUNNING, obtenido %s", job.Status)
	}
	if job.Progress != 50.0 {
		t.Errorf("esperado progreso 50.0, obtenido %.2f", job.Progress)
	}
}

// TestTask verifica la creación y gestión de tareas
func TestTask(t *testing.T) {
	now := time.Now()
	task := Task{
		ID:              "task-001",
		JobID:           "job-001",
		NodeID:          "node1",
		Operator:        OpMap,
		Partition:       0,
		TotalPartitions: 3,
		Status:          TaskStatusPending,
		AttemptNum:      0,
		CreatedAt:       now,
	}

	if task.ID != "task-001" {
		t.Errorf("esperado ID 'task-001', obtenido '%s'", task.ID)
	}
	if task.Status != TaskStatusPending {
		t.Errorf("esperado status PENDING, obtenido %s", task.Status)
	}
	if task.AttemptNum != 0 {
		t.Errorf("esperado AttemptNum 0, obtenido %d", task.AttemptNum)
	}

	// Simular asignación a worker
	task.WorkerID = "worker-1"
	task.Status = TaskStatusRunning
	task.StartedAt = &now
	task.AttemptNum = 1

	if task.WorkerID != "worker-1" {
		t.Errorf("esperado worker-1, obtenido %s", task.WorkerID)
	}
	if task.Status != TaskStatusRunning {
		t.Errorf("esperado status RUNNING, obtenido %s", task.Status)
	}
}

// TestTaskRetry verifica el mecanismo de reintentos
func TestTaskRetry(t *testing.T) {
	task := Task{
		ID:         "task-retry",
		JobID:      "job-001",
		Status:     TaskStatusPending,
		AttemptNum: 0,
	}

	// Simular fallo y reintento
	task.Status = TaskStatusFailed
	task.Error = "timeout"
	task.AttemptNum = 1

	if task.AttemptNum != 1 {
		t.Errorf("esperado intento 1, obtenido %d", task.AttemptNum)
	}

	// Segundo reintento
	task.Status = TaskStatusPending
	task.AttemptNum = 2

	if task.AttemptNum != 2 {
		t.Errorf("esperado intento 2, obtenido %d", task.AttemptNum)
	}

	// Tercer reintento (máximo)
	task.Status = TaskStatusFailed
	task.AttemptNum = 3

	if task.AttemptNum != 3 {
		t.Errorf("esperado intento 3, obtenido %d", task.AttemptNum)
	}
}

// TestWorkerInfo verifica la información de workers
func TestWorkerInfo(t *testing.T) {
	now := time.Now()
	worker := WorkerInfo{
		ID:            "worker-1",
		Address:       "192.168.1.100:8081",
		Status:        WorkerStatusUp,
		RegisteredAt:  now,
		LastHeartbeat: now,
		ActiveTasks:   5,
		TotalTasks:    10,
	}

	if worker.ID != "worker-1" {
		t.Errorf("esperado worker-1, obtenido %s", worker.ID)
	}
	if worker.Status != WorkerStatusUp {
		t.Errorf("esperado status UP, obtenido %s", worker.Status)
	}
	if worker.ActiveTasks != 5 {
		t.Errorf("esperado 5 tareas activas, obtenido %d", worker.ActiveTasks)
	}
	if worker.TotalTasks != 10 {
		t.Errorf("esperado 10 tareas totales, obtenido %d", worker.TotalTasks)
	}
}

// TestTaskTimeout verifica configuración de timeouts
func TestTaskTimeout(t *testing.T) {
	task := Task{
		ID:          "task-timeout",
		JobID:       "job-001",
		TimeoutSec:  30,
		MaxMemoryMB: 512,
	}

	if task.TimeoutSec != 30 {
		t.Errorf("esperado timeout 30s, obtenido %d", task.TimeoutSec)
	}
	if task.MaxMemoryMB != 512 {
		t.Errorf("esperado max memory 512MB, obtenido %d", task.MaxMemoryMB)
	}

	// Sin límites
	taskNoLimits := Task{
		ID:          "task-no-limits",
		JobID:       "job-001",
		TimeoutSec:  0,
		MaxMemoryMB: 0,
	}

	if taskNoLimits.TimeoutSec != 0 {
		t.Errorf("esperado sin timeout, obtenido %d", taskNoLimits.TimeoutSec)
	}
	if taskNoLimits.MaxMemoryMB != 0 {
		t.Errorf("esperado sin límite de memoria, obtenido %d", taskNoLimits.MaxMemoryMB)
	}
}

// TestTaskDuration verifica el cálculo de duración
func TestTaskDuration(t *testing.T) {
	startTime := time.Now()
	task := Task{
		ID:        "task-duration",
		JobID:     "job-001",
		Status:    TaskStatusRunning,
		StartedAt: &startTime,
	}

	// Simular 100ms de ejecución
	time.Sleep(100 * time.Millisecond)
	completedTime := time.Now()
	task.CompletedAt = &completedTime
	task.Status = TaskStatusCompleted
	task.DurationMs = completedTime.Sub(startTime).Milliseconds()

	if task.DurationMs < 100 {
		t.Errorf("esperado duración >= 100ms, obtenido %dms", task.DurationMs)
	}
	if task.DurationMs > 200 {
		t.Errorf("duración muy alta: %dms (esperado ~100ms)", task.DurationMs)
	}
}

// TestJobMetrics verifica las métricas de job
func TestJobMetrics(t *testing.T) {
	metrics := JobMetrics{
		JobID:          "job-001",
		TotalTasks:     10,
		CompletedTasks: 7,
		FailedTasks:    1,
		Duration:       5 * time.Second,
		Throughput:     1.4,
	}

	if metrics.CompletedTasks != 7 {
		t.Errorf("esperado 7 tareas completadas, obtenido %d", metrics.CompletedTasks)
	}
	if metrics.FailedTasks != 1 {
		t.Errorf("esperado 1 tarea fallida, obtenido %d", metrics.FailedTasks)
	}

	// Calcular progreso
	progress := float64(metrics.CompletedTasks) / float64(metrics.TotalTasks) * 100
	if progress != 70.0 {
		t.Errorf("esperado progreso 70%%, obtenido %.2f%%", progress)
	}
}

// TestDAGEdge verifica las aristas del DAG
func TestDAGEdge(t *testing.T) {
	edge := DAGEdge{
		From: "node1",
		To:   "node2",
	}

	if edge.From != "node1" {
		t.Errorf("esperado From='node1', obtenido '%s'", edge.From)
	}
	if edge.To != "node2" {
		t.Errorf("esperado To='node2', obtenido '%s'", edge.To)
	}
}

// TestTaskDependencies verifica las dependencias entre tareas
func TestTaskDependencies(t *testing.T) {
	task := Task{
		ID:           "task-with-deps",
		JobID:        "job-001",
		Dependencies: []string{"task-1", "task-2", "task-3"},
	}

	if len(task.Dependencies) != 3 {
		t.Errorf("esperado 3 dependencias, obtenido %d", len(task.Dependencies))
	}

	// Verificar que las dependencias se almacenan correctamente
	expectedDeps := map[string]bool{
		"task-1": true,
		"task-2": true,
		"task-3": true,
	}

	for _, dep := range task.Dependencies {
		if !expectedDeps[dep] {
			t.Errorf("dependencia inesperada: %s", dep)
		}
	}
}

// TestTaskPartitioning verifica el particionamiento de tareas
func TestTaskPartitioning(t *testing.T) {
	// Tarea con 4 particiones
	tasks := make([]Task, 4)
	for i := 0; i < 4; i++ {
		tasks[i] = Task{
			ID:              fmt.Sprintf("task-part-%d", i),
			JobID:           "job-001",
			Partition:       i,
			TotalPartitions: 4,
		}
	}

	// Verificar que todas las particiones están presentes
	for i, task := range tasks {
		if task.Partition != i {
			t.Errorf("esperado partición %d, obtenido %d", i, task.Partition)
		}
		if task.TotalPartitions != 4 {
			t.Errorf("esperado 4 particiones totales, obtenido %d", task.TotalPartitions)
		}
	}
}
