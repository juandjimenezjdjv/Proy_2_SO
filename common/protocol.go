package common

import "time"

// MessageType define los tipos de mensajes entre master y workers
type MessageType string

const (
	// Mensajes de registro y control
	MsgTypeRegisterWorker   MessageType = "REGISTER_WORKER"
	MsgTypeWorkerRegistered MessageType = "WORKER_REGISTERED"
	MsgTypeHeartbeat        MessageType = "HEARTBEAT"
	MsgTypeHeartbeatAck     MessageType = "HEARTBEAT_ACK"
	
	// Mensajes de asignación de tareas
	MsgTypeAssignTask    MessageType = "ASSIGN_TASK"
	MsgTypeTaskAssigned  MessageType = "TASK_ASSIGNED"
	
	// Mensajes de actualización de estado
	MsgTypeTaskUpdate    MessageType = "TASK_UPDATE"
	MsgTypeTaskCompleted MessageType = "TASK_COMPLETED"
	MsgTypeTaskFailed    MessageType = "TASK_FAILED"
)

// Message representa un mensaje genérico entre componentes
type Message struct {
	Type      MessageType            `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

// RegisterWorkerRequest es el mensaje de registro de un worker
type RegisterWorkerRequest struct {
	WorkerID string `json:"worker_id"`
	Address  string `json:"address"`
}

// RegisterWorkerResponse es la respuesta al registro de un worker
type RegisterWorkerResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	MasterID     string `json:"master_id"`
	HeartbeatSec int    `json:"heartbeat_sec"`
}

// HeartbeatRequest es el mensaje de heartbeat de un worker
type HeartbeatRequest struct {
	WorkerID    string `json:"worker_id"`
	ActiveTasks int    `json:"active_tasks"`
}

// HeartbeatResponse es la respuesta a un heartbeat
type HeartbeatResponse struct {
	Success   bool   `json:"success"`
	Timestamp time.Time `json:"timestamp"`
}

// AssignTaskRequest es el mensaje para asignar una tarea a un worker
type AssignTaskRequest struct {
	Task Task `json:"task"`
}

// AssignTaskResponse es la respuesta a la asignación de una tarea
type AssignTaskResponse struct {
	Success bool   `json:"success"`
	TaskID  string `json:"task_id"`
	Message string `json:"message"`
}

// TaskUpdateRequest es el mensaje de actualización de estado de una tarea
type TaskUpdateRequest struct {
	TaskID   string     `json:"task_id"`
	Status   TaskStatus `json:"status"`
	Progress float64    `json:"progress"`
	Error    string     `json:"error,omitempty"`
}

// TaskUpdateResponse es la respuesta a una actualización de tarea
type TaskUpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
