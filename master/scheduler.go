package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/juandjimenezjdjv/Proy_2_SO/common"
)

// Scheduler maneja la planificación y asignación de tareas del DAG
type Scheduler struct {
	master *Master
	mu     sync.RWMutex
}

// NewScheduler crea un nuevo planificador
func NewScheduler(master *Master) *Scheduler {
	return &Scheduler{
		master: master,
	}
}

// ScheduleJob analiza el DAG y crea las tareas para ejecutar
func (s *Scheduler) ScheduleJob(job *common.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Validar el DAG
	if err := s.validateDAG(job.DAG); err != nil {
		return fmt.Errorf("DAG inválido: %w", err)
	}

	// 2. Ordenamiento topológico (determinar orden de ejecución)
	sortedNodes, err := s.topologicalSort(job.DAG)
	if err != nil {
		return fmt.Errorf("error en ordenamiento topológico: %w", err)
	}

	// 3. Crear tareas según el orden
	tasks := s.createTasks(job.ID, sortedNodes, job.DAG)

	// 4. Asignar tareas a workers disponibles
	if err := s.assignTasks(job, tasks); err != nil {
		return fmt.Errorf("error asignando tareas: %w", err)
	}

	return nil
}

// validateDAG verifica que el DAG sea válido (sin ciclos, nodos válidos)
func (s *Scheduler) validateDAG(dag common.DAG) error {
	// Verificar que todos los nodos existen
	nodeMap := make(map[string]bool)
	for _, node := range dag.Nodes {
		if node.ID == "" {
			return fmt.Errorf("nodo sin ID")
		}
		if nodeMap[node.ID] {
			return fmt.Errorf("nodo duplicado: %s", node.ID)
		}
		nodeMap[node.ID] = true
	}

	// Verificar que las aristas apuntan a nodos válidos
	for _, edge := range dag.Edges {
		if !nodeMap[edge.From] {
			return fmt.Errorf("arista desde nodo inexistente: %s", edge.From)
		}
		if !nodeMap[edge.To] {
			return fmt.Errorf("arista hacia nodo inexistente: %s", edge.To)
		}
	}

	return nil
}

// topologicalSort realiza ordenamiento topológico usando algoritmo de Kahn
func (s *Scheduler) topologicalSort(dag common.DAG) ([]common.DAGNode, error) {
	// Construir grafo de dependencias
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)

	// Inicializar grado de entrada
	for _, node := range dag.Nodes {
		inDegree[node.ID] = 0
	}

	// Construir lista de adyacencia y calcular grados de entrada
	for _, edge := range dag.Edges {
		adjList[edge.From] = append(adjList[edge.From], edge.To)
		inDegree[edge.To]++
	}

	// Cola con nodos sin dependencias (grado 0)
	queue := []string{}
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	// Procesar nodos
	result := []common.DAGNode{}
	nodeMap := make(map[string]common.DAGNode)
	for _, node := range dag.Nodes {
		nodeMap[node.ID] = node
	}

	processed := 0
	for len(queue) > 0 {
		// Sacar primer elemento
		currentID := queue[0]
		queue = queue[1:]

		result = append(result, nodeMap[currentID])
		processed++

		// Reducir grado de entrada de vecinos
		for _, neighbor := range adjList[currentID] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Si no procesamos todos los nodos, hay un ciclo
	if processed != len(dag.Nodes) {
		return nil, fmt.Errorf("DAG contiene ciclos")
	}

	return result, nil
}

// createTasks convierte nodos del DAG en tareas ejecutables
func (s *Scheduler) createTasks(jobID string, nodes []common.DAGNode, dag common.DAG) []*common.Task {
	tasks := make([]*common.Task, 0, len(nodes))

	for i, node := range nodes {
		// Determinar dependencias (tareas que deben completarse antes)
		dependencies := s.findDependencies(node.ID, dag)

		// Determinar número de particiones (por defecto 1, configurable)
		numPartitions := node.Partitions
		if numPartitions == 0 {
			numPartitions = 1
		}

		// Crear una tarea por cada partición
		for p := 0; p < numPartitions; p++ {
			taskID := fmt.Sprintf("%s-task-%d-part-%d", jobID, i, p)

			// Extraer timeout y límite de memoria de los params si existen
			timeoutSec := 0
			maxMemoryMB := int64(0)
			if node.Params != nil {
				if t, ok := node.Params["timeout_sec"].(float64); ok {
					timeoutSec = int(t)
				}
				if m, ok := node.Params["max_memory_mb"].(float64); ok {
					maxMemoryMB = int64(m)
				}
			}

			task := &common.Task{
				ID:              taskID,
				JobID:           jobID,
				NodeID:          node.ID,
				Operator:        node.Operator,
				InputPaths:      node.InputPaths,
				OutputPath:      node.OutputPath,
				Partition:       p,
				TotalPartitions: numPartitions,
				Dependencies:    dependencies,
				Params:          node.Params, // Agregar parámetros del nodo
				Status:          common.TaskPending,
				AttemptNum:      0,
				CreatedAt:       time.Now(),
				TimeoutSec:      timeoutSec,
				MaxMemoryMB:     maxMemoryMB,
			}
			tasks = append(tasks, task)
		}
	}

	return tasks
}

// findDependencies encuentra los IDs de tareas de las que depende un nodo
func (s *Scheduler) findDependencies(nodeID string, dag common.DAG) []string {
	deps := []string{}
	for _, edge := range dag.Edges {
		if edge.To == nodeID {
			deps = append(deps, edge.From)
		}
	}
	return deps
}

// assignTasks asigna tareas a workers disponibles usando round-robin
func (s *Scheduler) assignTasks(job *common.Job, tasks []*common.Task) error {
	s.master.workersMutex.RLock()
	availableWorkers := s.getHealthyWorkers()
	s.master.workersMutex.RUnlock()

	if len(availableWorkers) == 0 {
		return fmt.Errorf("no hay workers disponibles")
	}

	// Guardar tareas en el job
	job.Tasks = tasks

	// Asignar usando balanceo por carga (menos tareas activas primero)
	for _, task := range tasks {
		// Solo asignar tareas sin dependencias en esta ronda
		if len(task.Dependencies) == 0 {
			// Encontrar worker con menos carga
			minLoad := int(^uint(0) >> 1) // Max int
			var selectedWorker *common.WorkerInfo

			for i := range availableWorkers {
				if availableWorkers[i].ActiveTasks < minLoad {
					minLoad = availableWorkers[i].ActiveTasks
					selectedWorker = &availableWorkers[i]
				}
			}

			if selectedWorker != nil {
				task.WorkerID = selectedWorker.ID
				task.Status = common.TaskAssigned
				selectedWorker.ActiveTasks++ // Incrementar para siguiente asignación
			}
		}

		// Guardar tarea en el map global del master
		s.master.tasksMutex.Lock()
		s.master.tasks[task.ID] = task
		s.master.tasksMutex.Unlock()
	}

	// Actualizar job status
	s.master.jobsMutex.Lock()
	if j, exists := s.master.jobs[job.ID]; exists {
		j.Status = common.JobStatusRunning
		j.StartedAt = time.Now()
		j.Tasks = tasks
	}
	s.master.jobsMutex.Unlock()

	return nil
}

// getHealthyWorkers retorna lista de workers activos
func (s *Scheduler) getHealthyWorkers() []common.WorkerInfo {
	workers := []common.WorkerInfo{}
	for _, worker := range s.master.workers {
		if worker.Status == common.WorkerUp {
			workers = append(workers, *worker)
		}
	}
	return workers
}

// ReassignFailedTask reasigna una tarea fallida a otro worker
func (s *Scheduler) ReassignFailedTask(task *common.Task) error {
	s.master.workersMutex.RLock()
	availableWorkers := s.getHealthyWorkers()
	s.master.workersMutex.RUnlock()

	if len(availableWorkers) == 0 {
		return fmt.Errorf("no hay workers disponibles para reasignar")
	}

	oldWorkerID := task.WorkerID
	var selectedWorker *common.WorkerInfo

	// Intentar encontrar worker diferente al que falló, preferir el menos cargado
	minLoad := int(^uint(0) >> 1) // Max int
	for _, worker := range availableWorkers {
		if worker.ID != oldWorkerID {
			if worker.ActiveTasks < minLoad {
				minLoad = worker.ActiveTasks
				selectedWorker = &worker
			}
		}
	}

	// Si no hay otro worker disponible, usar cualquiera
	if selectedWorker == nil {
		// Elegir el menos cargado
		for _, worker := range availableWorkers {
			if worker.ActiveTasks < minLoad {
				minLoad = worker.ActiveTasks
				selectedWorker = &worker
			}
		}
	}

	if selectedWorker == nil {
		return fmt.Errorf("no se pudo seleccionar worker para reasignación")
	}

	task.WorkerID = selectedWorker.ID
	task.Status = common.TaskAssigned
	task.Error = "" // Limpiar error anterior

	s.master.logger.Info("Tarea %s reasignada de %s a %s (intento %d)",
		task.ID, oldWorkerID, selectedWorker.ID, task.AttemptNum)

	return nil
}
