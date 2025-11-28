package main

import (
	"testing"
	"time"

	"github.com/juandjimenezjdjv/Proy_2_SO/common"
)

// TestTopologicalSort prueba el ordenamiento topológico del DAG
func TestTopologicalSort(t *testing.T) {
	master := &Master{
		workers: make(map[string]*common.WorkerInfo),
		jobs:    make(map[string]*common.Job),
		tasks:   make(map[string]*common.Task),
	}

	scheduler := NewScheduler(master)

	// DAG simple: read -> map -> reduce
	dag := common.DAG{
		Nodes: []common.DAGNode{
			{ID: "read", Operator: common.OpReadCSV},
			{ID: "map", Operator: common.OpMap},
			{ID: "reduce", Operator: common.OpReduceByKey},
		},
		Edges: []common.DAGEdge{
			{From: "read", To: "map"},
			{From: "map", To: "reduce"},
		},
	}

	sorted, err := scheduler.topologicalSort(dag)
	if err != nil {
		t.Fatalf("error en ordenamiento topológico: %v", err)
	}

	if len(sorted) != 3 {
		t.Errorf("esperado 3 nodos, obtenido %d", len(sorted))
	}

	// Verificar orden: read debe estar antes de map, map antes de reduce
	nodeOrder := make(map[string]int)
	for i, node := range sorted {
		nodeOrder[node.ID] = i
	}

	if nodeOrder["read"] >= nodeOrder["map"] {
		t.Error("read debe estar antes de map")
	}
	if nodeOrder["map"] >= nodeOrder["reduce"] {
		t.Error("map debe estar antes de reduce")
	}
}

// TestTopologicalSortComplex prueba ordenamiento con DAG más complejo
func TestTopologicalSortComplex(t *testing.T) {
	master := &Master{
		workers: make(map[string]*common.WorkerInfo),
		jobs:    make(map[string]*common.Job),
		tasks:   make(map[string]*common.Task),
	}

	scheduler := NewScheduler(master)

	// DAG con múltiples ramas:
	//     read1 -> map1 \
	//                    -> join -> reduce
	//     read2 -> map2 /
	dag := common.DAG{
		Nodes: []common.DAGNode{
			{ID: "read1", Operator: common.OpReadCSV},
			{ID: "read2", Operator: common.OpReadCSV},
			{ID: "map1", Operator: common.OpMap},
			{ID: "map2", Operator: common.OpMap},
			{ID: "join", Operator: common.OpJoin},
			{ID: "reduce", Operator: common.OpReduceByKey},
		},
		Edges: []common.DAGEdge{
			{From: "read1", To: "map1"},
			{From: "read2", To: "map2"},
			{From: "map1", To: "join"},
			{From: "map2", To: "join"},
			{From: "join", To: "reduce"},
		},
	}

	sorted, err := scheduler.topologicalSort(dag)
	if err != nil {
		t.Fatalf("error en ordenamiento topológico: %v", err)
	}

	if len(sorted) != 6 {
		t.Errorf("esperado 6 nodos, obtenido %d", len(sorted))
	}

	// Verificar que las dependencias se respetan
	nodeOrder := make(map[string]int)
	for i, node := range sorted {
		nodeOrder[node.ID] = i
	}

	// read1 debe estar antes de map1
	if nodeOrder["read1"] >= nodeOrder["map1"] {
		t.Error("read1 debe estar antes de map1")
	}

	// read2 debe estar antes de map2
	if nodeOrder["read2"] >= nodeOrder["map2"] {
		t.Error("read2 debe estar antes de map2")
	}

	// map1 y map2 deben estar antes de join
	if nodeOrder["map1"] >= nodeOrder["join"] {
		t.Error("map1 debe estar antes de join")
	}
	if nodeOrder["map2"] >= nodeOrder["join"] {
		t.Error("map2 debe estar antes de join")
	}

	// join debe estar antes de reduce
	if nodeOrder["join"] >= nodeOrder["reduce"] {
		t.Error("join debe estar antes de reduce")
	}
}

// TestDAGValidation prueba la validación de DAG
func TestDAGValidation(t *testing.T) {
	master := &Master{
		workers: make(map[string]*common.WorkerInfo),
		jobs:    make(map[string]*common.Job),
		tasks:   make(map[string]*common.Task),
	}

	scheduler := NewScheduler(master)

	// DAG válido
	validDAG := common.DAG{
		Nodes: []common.DAGNode{
			{ID: "node1", Operator: common.OpReadCSV},
			{ID: "node2", Operator: common.OpMap},
		},
		Edges: []common.DAGEdge{
			{From: "node1", To: "node2"},
		},
	}

	if err := scheduler.validateDAG(validDAG); err != nil {
		t.Errorf("DAG válido marcado como inválido: %v", err)
	}

	// DAG con nodo sin ID
	invalidDAG1 := common.DAG{
		Nodes: []common.DAGNode{
			{ID: "", Operator: common.OpReadCSV},
		},
	}

	if err := scheduler.validateDAG(invalidDAG1); err == nil {
		t.Error("esperado error para nodo sin ID")
	}

	// DAG con nodo duplicado
	invalidDAG2 := common.DAG{
		Nodes: []common.DAGNode{
			{ID: "dup", Operator: common.OpReadCSV},
			{ID: "dup", Operator: common.OpMap},
		},
	}

	if err := scheduler.validateDAG(invalidDAG2); err == nil {
		t.Error("esperado error para nodo duplicado")
	}

	// DAG con arista a nodo inexistente
	invalidDAG3 := common.DAG{
		Nodes: []common.DAGNode{
			{ID: "node1", Operator: common.OpReadCSV},
		},
		Edges: []common.DAGEdge{
			{From: "node1", To: "nonexistent"},
		},
	}

	if err := scheduler.validateDAG(invalidDAG3); err == nil {
		t.Error("esperado error para arista a nodo inexistente")
	}
}

// TestCycleDetection prueba detección de ciclos en DAG
func TestCycleDetection(t *testing.T) {
	master := &Master{
		workers: make(map[string]*common.WorkerInfo),
		jobs:    make(map[string]*common.Job),
		tasks:   make(map[string]*common.Task),
	}

	scheduler := NewScheduler(master)

	// DAG con ciclo: A -> B -> C -> A
	cyclicDAG := common.DAG{
		Nodes: []common.DAGNode{
			{ID: "A", Operator: common.OpMap},
			{ID: "B", Operator: common.OpMap},
			{ID: "C", Operator: common.OpMap},
		},
		Edges: []common.DAGEdge{
			{From: "A", To: "B"},
			{From: "B", To: "C"},
			{From: "C", To: "A"}, // Ciclo!
		},
	}

	_, err := scheduler.topologicalSort(cyclicDAG)
	if err == nil {
		t.Error("esperado error para DAG con ciclo")
	}

	if err != nil && err.Error() != "DAG contiene ciclos" {
		t.Errorf("mensaje de error incorrecto: %v", err)
	}
}

// TestCreateTasks prueba la creación de tareas desde DAG
func TestCreateTasks(t *testing.T) {
	master := &Master{
		workers: make(map[string]*common.WorkerInfo),
		jobs:    make(map[string]*common.Job),
		tasks:   make(map[string]*common.Task),
	}

	scheduler := NewScheduler(master)

	dag := common.DAG{
		Nodes: []common.DAGNode{
			{
				ID:         "read",
				Operator:   common.OpReadCSV,
				Partitions: 2,
			},
			{
				ID:         "map",
				Operator:   common.OpMap,
				Partitions: 1,
			},
		},
		Edges: []common.DAGEdge{
			{From: "read", To: "map"},
		},
	}

	sorted, _ := scheduler.topologicalSort(dag)
	tasks := scheduler.createTasks("job-123", sorted, dag)

	// Debe crear 3 tareas: 2 de read (particiones) + 1 de map
	expectedTasks := 3
	if len(tasks) != expectedTasks {
		t.Errorf("esperado %d tareas, obtenido %d", expectedTasks, len(tasks))
	}

	// Verificar que las tareas tienen IDs únicos
	taskIDs := make(map[string]bool)
	for _, task := range tasks {
		if taskIDs[task.ID] {
			t.Errorf("ID de tarea duplicado: %s", task.ID)
		}
		taskIDs[task.ID] = true
	}

	// Verificar que las tareas de map tienen dependencias de read
	for _, task := range tasks {
		if task.NodeID == "map" {
			if len(task.Dependencies) != 1 {
				t.Errorf("tarea map debe tener 1 dependencia, tiene %d", len(task.Dependencies))
			}
			if task.Dependencies[0] != "read" {
				t.Errorf("dependencia incorrecta: esperado 'read', obtenido '%s'", task.Dependencies[0])
			}
		}
	}
}

// TestTaskAssignment prueba la asignación de tareas a workers
func TestTaskAssignment(t *testing.T) {
	master := &Master{
		workers: make(map[string]*common.WorkerInfo),
		jobs:    make(map[string]*common.Job),
		tasks:   make(map[string]*common.Task),
	}

	// Agregar workers
	master.workers["worker1"] = &common.WorkerInfo{
		ID:          "worker1",
		Status:      common.WorkerUp,
		ActiveTasks: 0,
	}
	master.workers["worker2"] = &common.WorkerInfo{
		ID:          "worker2",
		Status:      common.WorkerUp,
		ActiveTasks: 0,
	}

	scheduler := NewScheduler(master)

	// Crear job simple
	job := &common.Job{
		ID:   "job-assign",
		Name: "test-assignment",
		DAG: common.DAG{
			Nodes: []common.DAGNode{
				{ID: "task1", Operator: common.OpMap, Partitions: 1},
				{ID: "task2", Operator: common.OpMap, Partitions: 1},
			},
		},
		Status:      common.JobStatusAccepted,
		SubmittedAt: time.Now(),
	}

	master.jobs[job.ID] = job

	err := scheduler.ScheduleJob(job)
	if err != nil {
		t.Fatalf("error asignando tareas: %v", err)
	}

	// Verificar que las tareas fueron asignadas
	if len(job.Tasks) == 0 {
		t.Error("no se crearon tareas")
	}

	// Verificar que todas las tareas sin dependencias tienen worker asignado
	for _, task := range job.Tasks {
		if len(task.Dependencies) == 0 && task.WorkerID == "" {
			t.Errorf("tarea %s sin dependencias no tiene worker asignado", task.ID)
		}
	}
}

// TestTaskAssignmentBalancing prueba balanceo de carga
func TestTaskAssignmentBalancing(t *testing.T) {
	master := &Master{
		workers: make(map[string]*common.WorkerInfo),
		jobs:    make(map[string]*common.Job),
		tasks:   make(map[string]*common.Task),
	}

	// Worker1 con más carga
	master.workers["worker1"] = &common.WorkerInfo{
		ID:          "worker1",
		Status:      common.WorkerUp,
		ActiveTasks: 5,
	}

	// Worker2 con menos carga
	master.workers["worker2"] = &common.WorkerInfo{
		ID:          "worker2",
		Status:      common.WorkerUp,
		ActiveTasks: 1,
	}

	scheduler := NewScheduler(master)

	// Crear job
	job := &common.Job{
		ID:   "job-balancing",
		Name: "test-balancing",
		DAG: common.DAG{
			Nodes: []common.DAGNode{
				{ID: "task1", Operator: common.OpMap, Partitions: 1},
			},
		},
		Status: common.JobStatusAccepted,
	}

	master.jobs[job.ID] = job

	err := scheduler.ScheduleJob(job)
	if err != nil {
		t.Fatalf("error en scheduling: %v", err)
	}

	// La tarea debe asignarse al worker con menor carga (worker2)
	if len(job.Tasks) > 0 {
		assignedWorker := job.Tasks[0].WorkerID
		if assignedWorker != "worker2" {
			t.Errorf("esperado asignación a worker2 (menos cargado), obtenido %s", assignedWorker)
		}
	}
}

// TestNoWorkersAvailable prueba comportamiento sin workers
func TestNoWorkersAvailable(t *testing.T) {
	master := &Master{
		workers: make(map[string]*common.WorkerInfo),
		jobs:    make(map[string]*common.Job),
		tasks:   make(map[string]*common.Task),
	}

	scheduler := NewScheduler(master)

	job := &common.Job{
		ID:   "job-no-workers",
		Name: "test-no-workers",
		DAG: common.DAG{
			Nodes: []common.DAGNode{
				{ID: "task1", Operator: common.OpMap},
			},
		},
		Status: common.JobStatusAccepted,
	}

	master.jobs[job.ID] = job

	err := scheduler.ScheduleJob(job)
	if err == nil {
		t.Error("esperado error cuando no hay workers disponibles")
	}
}

// TestReassignFailedTask prueba reasignación de tareas fallidas
func TestReassignFailedTask(t *testing.T) {
	master := &Master{
		workers: make(map[string]*common.WorkerInfo),
		jobs:    make(map[string]*common.Job),
		tasks:   make(map[string]*common.Task),
		logger:  common.NewLogger("MASTER", common.LogLevelInfo),
	}

	// Agregar workers
	master.workers["worker1"] = &common.WorkerInfo{
		ID:          "worker1",
		Status:      common.WorkerUp,
		ActiveTasks: 3,
	}
	master.workers["worker2"] = &common.WorkerInfo{
		ID:          "worker2",
		Status:      common.WorkerUp,
		ActiveTasks: 1,
	}

	scheduler := NewScheduler(master)

	// Crear tarea fallida asignada a worker1
	task := &common.Task{
		ID:         "failed-task",
		JobID:      "job-1",
		WorkerID:   "worker1",
		Status:     common.TaskFailed,
		AttemptNum: 1,
		Error:      "timeout",
	}

	err := scheduler.ReassignFailedTask(task)
	if err != nil {
		t.Fatalf("error reasignando tarea: %v", err)
	}

	// Verificar que se reasignó a otro worker
	if task.WorkerID == "worker1" {
		t.Error("tarea no se reasignó a diferente worker")
	}

	// Debe asignarse a worker2 (menos cargado)
	if task.WorkerID != "worker2" {
		t.Errorf("esperado reasignación a worker2, obtenido %s", task.WorkerID)
	}

	// Verificar que el status cambió
	if task.Status != common.TaskAssigned {
		t.Errorf("status incorrecto después de reasignación: %s", task.Status)
	}

	// Verificar que el error se limpió
	if task.Error != "" {
		t.Errorf("error no se limpió: %s", task.Error)
	}
}

// TestReassignWithSingleWorker prueba reasignación con un solo worker
func TestReassignWithSingleWorker(t *testing.T) {
	master := &Master{
		workers: make(map[string]*common.WorkerInfo),
		jobs:    make(map[string]*common.Job),
		tasks:   make(map[string]*common.Task),
		logger:  common.NewLogger("MASTER", common.LogLevelInfo),
	}

	// Solo un worker disponible
	master.workers["worker1"] = &common.WorkerInfo{
		ID:          "worker1",
		Status:      common.WorkerUp,
		ActiveTasks: 2,
	}

	scheduler := NewScheduler(master)

	task := &common.Task{
		ID:         "failed-task",
		JobID:      "job-1",
		WorkerID:   "worker1",
		Status:     common.TaskFailed,
		AttemptNum: 2,
	}

	err := scheduler.ReassignFailedTask(task)
	if err != nil {
		t.Fatalf("error reasignando: %v", err)
	}

	// Debe reasignarse al mismo worker (único disponible)
	if task.WorkerID != "worker1" {
		t.Errorf("esperado worker1, obtenido %s", task.WorkerID)
	}
}

// TestFindDependencies prueba búsqueda de dependencias
func TestFindDependencies(t *testing.T) {
	master := &Master{
		workers: make(map[string]*common.WorkerInfo),
		jobs:    make(map[string]*common.Job),
		tasks:   make(map[string]*common.Task),
	}

	scheduler := NewScheduler(master)

	dag := common.DAG{
		Edges: []common.DAGEdge{
			{From: "A", To: "B"},
			{From: "C", To: "B"},
			{From: "B", To: "D"},
		},
	}

	// B depende de A y C
	depsB := scheduler.findDependencies("B", dag)
	if len(depsB) != 2 {
		t.Errorf("esperado 2 dependencias para B, obtenido %d", len(depsB))
	}

	// D depende solo de B
	depsD := scheduler.findDependencies("D", dag)
	if len(depsD) != 1 {
		t.Errorf("esperado 1 dependencia para D, obtenido %d", len(depsD))
	}
	if depsD[0] != "B" {
		t.Errorf("esperado dependencia 'B', obtenido '%s'", depsD[0])
	}

	// A no tiene dependencias
	depsA := scheduler.findDependencies("A", dag)
	if len(depsA) != 0 {
		t.Errorf("esperado 0 dependencias para A, obtenido %d", len(depsA))
	}
}

// TestGetHealthyWorkers prueba filtrado de workers activos
func TestGetHealthyWorkers(t *testing.T) {
	master := &Master{
		workers: make(map[string]*common.WorkerInfo),
		jobs:    make(map[string]*common.Job),
		tasks:   make(map[string]*common.Task),
	}

	master.workers["worker1"] = &common.WorkerInfo{
		ID:     "worker1",
		Status: common.WorkerUp,
	}
	master.workers["worker2"] = &common.WorkerInfo{
		ID:     "worker2",
		Status: common.WorkerDown, // Worker caído
	}
	master.workers["worker3"] = &common.WorkerInfo{
		ID:     "worker3",
		Status: common.WorkerUp,
	}

	scheduler := NewScheduler(master)
	healthy := scheduler.getHealthyWorkers()

	// Solo worker1 y worker3 deben estar en la lista
	if len(healthy) != 2 {
		t.Errorf("esperado 2 workers healthy, obtenido %d", len(healthy))
	}

	healthyIDs := make(map[string]bool)
	for _, w := range healthy {
		healthyIDs[w.ID] = true
	}

	if !healthyIDs["worker1"] || !healthyIDs["worker3"] {
		t.Error("workers healthy incorrectos")
	}
	if healthyIDs["worker2"] {
		t.Error("worker2 (DOWN) no debería estar en lista healthy")
	}
}

// TestTaskWithTimeoutAndMemoryLimits prueba creación de tareas con límites
func TestTaskWithTimeoutAndMemoryLimits(t *testing.T) {
	master := &Master{
		workers: make(map[string]*common.WorkerInfo),
		jobs:    make(map[string]*common.Job),
		tasks:   make(map[string]*common.Task),
	}

	scheduler := NewScheduler(master)

	dag := common.DAG{
		Nodes: []common.DAGNode{
			{
				ID:       "limited-task",
				Operator: common.OpMap,
				Params: map[string]interface{}{
					"timeout_sec":   float64(60),
					"max_memory_mb": float64(512),
				},
			},
		},
	}

	sorted, _ := scheduler.topologicalSort(dag)
	tasks := scheduler.createTasks("job-limits", sorted, dag)

	if len(tasks) == 0 {
		t.Fatal("no se crearon tareas")
	}

	task := tasks[0]
	if task.TimeoutSec != 60 {
		t.Errorf("esperado timeout 60s, obtenido %d", task.TimeoutSec)
	}
	if task.MaxMemoryMB != 512 {
		t.Errorf("esperado max memory 512MB, obtenido %d", task.MaxMemoryMB)
	}
}

// TestScheduleJobComplete prueba flujo completo de scheduling
func TestScheduleJobComplete(t *testing.T) {
	master := &Master{
		workers: make(map[string]*common.WorkerInfo),
		jobs:    make(map[string]*common.Job),
		tasks:   make(map[string]*common.Task),
	}

	// Agregar workers
	master.workers["worker1"] = &common.WorkerInfo{
		ID:          "worker1",
		Status:      common.WorkerUp,
		ActiveTasks: 0,
	}

	scheduler := NewScheduler(master)

	job := &common.Job{
		ID:   "complete-job",
		Name: "test-complete",
		DAG: common.DAG{
			Nodes: []common.DAGNode{
				{ID: "read", Operator: common.OpReadCSV, Partitions: 2},
				{ID: "map", Operator: common.OpMap, Partitions: 1},
			},
			Edges: []common.DAGEdge{
				{From: "read", To: "map"},
			},
		},
		Status:      common.JobStatusAccepted,
		Parallelism: 3,
		SubmittedAt: time.Now(),
	}

	master.jobs[job.ID] = job

	err := scheduler.ScheduleJob(job)
	if err != nil {
		t.Fatalf("error en ScheduleJob: %v", err)
	}

	// Verificar que el job cambió a RUNNING
	if job.Status != common.JobStatusRunning {
		t.Errorf("esperado status RUNNING, obtenido %s", job.Status)
	}

	// Verificar que se crearon las tareas correctas
	if len(job.Tasks) != 3 { // 2 read + 1 map
		t.Errorf("esperado 3 tareas, obtenido %d", len(job.Tasks))
	}

	// Verificar que las tareas se guardaron en master.tasks
	for _, task := range job.Tasks {
		if _, exists := master.tasks[task.ID]; !exists {
			t.Errorf("tarea %s no está en master.tasks", task.ID)
		}
	}
}
