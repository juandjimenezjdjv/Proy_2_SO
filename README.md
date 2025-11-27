# Mini-Spark: Motor de Procesamiento Distribuido

Sistema de procesamiento distribuido batch implementado en Go desde cero, sin frameworks como Spark o Flink.

## Inicio Rápido

```bash
# Iniciar el cluster (1 master + 3 workers)
make up

# Ver estado del cluster
make status

# Ver logs
make logs

# Detener el cluster
make down
```

## Estructura del Proyecto

```
├── common/             # Código compartido (tipos, protocolos, config, logger)
├── master/             # Coordinador principal del sistema
├── worker/             # Nodos de ejecución de tareas
├── examples/           # Ejemplos de jobs (wordcount, join)
├── data/               # Datos de entrada
├── results/            # Resultados procesados
├── storage/            # Estado persistente
├── docker-compose.yml  # Configuración del cluster
├── Makefile            # Automatización
└── go.mod              # Módulo Go
```

## Arquitectura

### Componentes

**1. Master (Coordinador)**
- Registro de workers y heartbeats
- Recepción y gestión de jobs
- Planificador de tareas
- API REST HTTP/JSON

**2. Workers (Ejecutores)**
- Ejecutan tareas asignadas
- Heartbeats periódicos al master
- Reporte de estado de tareas
- Manejo de particiones

**3. Common (Compartido)**
- Tipos: Job, Task, DAG, Worker
- Protocolos de comunicación
- Sistema de logging
- Configuración

## API REST

### Endpoints del Master

```http
# Health check
GET /health

# Registrar worker
POST /api/v1/workers/register

# Heartbeat
POST /api/v1/workers/heartbeat

# Crear job
POST /api/v1/jobs

# Listar jobs
GET /api/v1/jobs

# Obtener job por ID
GET /api/v1/jobs/{id}

# Actualizar tarea
POST /api/v1/tasks/update
```

## Configuración

Variables de entorno disponibles:

| Variable | Default | Descripción |
|----------|---------|-------------|
| `MASTER_HOST` | localhost | Host del master |
| `MASTER_PORT` | 8080 | Puerto del master |
| `HEARTBEAT_SEC` | 2 | Intervalo de heartbeats |
| `HEARTBEAT_TIMEOUT_SEC` | 10 | Timeout para marcar worker DOWN |
| `WORKER_THREADS` | 4 | Threads por worker |
| `MAX_RETRIES` | 3 | Reintentos máximos |
| `MAX_MEMORY_MB` | 100 | Límite de memoria para cache (MB) |
| `LOG_LEVEL` | INFO | DEBUG, INFO, WARN, ERROR |

## Despliegue con Docker

### Iniciar Cluster

```bash
# Construir e iniciar
make up

# Ver estado
make status

# Logs en tiempo real
make logs

# Detener
make down
```

### Verificar Funcionamiento

```bash
# Health check
curl http://localhost:8080/health

# Workers registrados
curl http://localhost:8080/api/v1/workers

# Listar jobs
curl http://localhost:8080/api/v1/jobs
```

### Arquitectura Docker

```
┌────────────────────────────────────┐
│    Docker Network (minispark-net)  │
│                                    │
│    Master :8080 (HTTP API)         │
│       ↕ ↕ ↕                        │
│    Worker1  Worker2  Worker3       │
│    (4 hilos cada uno)              │
│                                    │
│ Volúmenes:                         │
│ • ./data → datos compartidos       │
│ • ./results → resultados           │
│ • ./storage → estado persistente   │
└────────────────────────────────────┘
```

## Desarrollo Local

### Compilar

```bash
# Compilar todos
make build

# Compilar master
make build-master

# Compilar worker
make build-worker
```

### Ejecutar Manualmente

```bash
# Terminal 1 - Master
export MASTER_HOST=localhost
export MASTER_PORT=8080
go run master/main.go

# Terminal 2 - Worker
export MASTER_HOST=localhost
export MASTER_PORT=8080
go run worker/main.go
```

## Testing

```bash
# Pruebas unitarias
make test

# Pruebas de integración
make test-integration

# Formatear código
make fmt

# Linting
make lint
```

## Comandos Make Disponibles

| Comando | Descripción |
|---------|-------------|
| `make up` | Inicia el cluster |
| `make down` | Detiene el cluster |
| `make restart` | Reinicia el cluster |
| `make logs` | Ver logs de todos los contenedores |
| `make logs-master` | Ver logs del master |
| `make logs-workers` | Ver logs de workers |
| `make status` | Estado del cluster |
| `make build` | Compilar binarios localmente |
| `make test` | Ejecutar pruebas |
| `make clean` | Limpieza |
| `make demo` | Demo interactivo |

## Roadmap

### ✅ Completado
- [x] Infraestructura base
- [x] Common package
- [x] Master con registro y heartbeats
- [x] Workers con heartbeats
- [x] Docker Compose y Makefile
- [x] Cliente CLI completo
- [x] Planificador de DAG con ordenamiento topológico
- [x] Operadores: read_csv, map, filter, **flat_map**, **aggregate** (count/sum/avg/min/max), reduce_by_key, join
- [x] Lectura/escritura CSV con particionamiento
- [x] Sistema de particiones hash-based
- [x] **Cache en memoria + spill a disco**
- [x] Tolerancia a fallos (detección, reintentos, reasignación)
- [x] **Persistencia del estado (auto-save)**
- [x] **Métricas de CPU/memoria por worker**
- [x] Balanceo de carga dinámico
- [x] Suite de pruebas y ejemplos

### 📋 Opcional/Futuro
- [ ] Operador aggregate explícito
- [ ] Shuffle como operador standalone
- [ ] Lectura/escritura JSONL
- [ ] Límites de tiempo/memoria por tarea
- [ ] Benchmarks formales con 1M+ registros
- [ ] Pruebas unitarias automatizadas
- [ ] Video demo y documentación de arquitectura

## Tecnologías

- **Lenguaje**: Go 1.21+
- **Comunicación**: HTTP/1.1 + JSON
- **Concurrencia**: Goroutines + Channels + Mutexes
- **Containerización**: Docker + Docker Compose
- **Sin dependencias externas** (solo stdlib)

## Uso de IA

Este proyecto utiliza GitHub Copilot como apoyo para documentación, estructuras base y patrones de diseño. Todo el código ha sido revisado y adaptado por el equipo.

## Equipo

- **Proyecto**: Mini-Spark - Motor de Procesamiento Distribuido
- **Curso**: Principios de Sistemas Operativos
- **Institución**: TEC Costa Rica
- **Profesor**: Kenneth Obando Rodríguez

## Licencia

Proyecto académico - TEC Costa Rica
