# Mini-Spark: Motor de Procesamiento Distribuido

Sistema de procesamiento distribuido batch implementado en Go desde cero, sin usar frameworks como Spark o Flink.

## Estructura del Proyecto

```
├── common/          # Código compartido (tipos, protocolos, utilidades)
├── master/          # Coordinador principal
├── worker/          # Nodos de ejecución
├── client/          # CLI para interactuar con el sistema
├── docs/            # Documentación de arquitectura
├── scripts/         # Scripts de build y despliegue
├── testdata/        # Datasets para pruebas
├── go.mod           # Definición del módulo Go
└── README.md        # Este archivo
```

## Componentes Implementados

### 1. Common Package (`/common`)
Código compartido entre todos los componentes:

- **types.go**: Definiciones de tipos principales
  - `Job`, `DAG`, `Task`: Estructuras de trabajos y tareas
  - `WorkerInfo`: Información de workers registrados
  - Estados: `JobStatus`, `TaskStatus`, `WorkerStatus`
  - Operadores: `OperatorType` (map, filter, reduce, etc.)

- **protocol.go**: Protocolos de comunicación
  - Mensajes de registro y control
  - Mensajes de heartbeat
  - Mensajes de asignación de tareas
  - Actualizaciones de estado

- **logger.go**: Sistema de logging estructurado
  - Niveles: DEBUG, INFO, WARN, ERROR
  - Formato con timestamp y componente

- **config.go**: Configuración del sistema
  - Variables de entorno con valores por defecto
  - Configuración de master, workers y almacenamiento

### 2. Master (`/master`)
Coordinador principal del sistema:

**Funcionalidades implementadas:**
- ✅ Servidor HTTP en puerto configurable
- ✅ Registro de workers (`POST /api/v1/workers/register`)
- ✅ Sistema de heartbeats (`POST /api/v1/workers/heartbeat`)
- ✅ Recepción de jobs (`POST /api/v1/jobs`)
- ✅ Consulta de jobs (`GET /api/v1/jobs`, `GET /api/v1/jobs/{id}`)
- ✅ Actualización de tareas (`POST /api/v1/tasks/update`)
- ✅ Health check (`GET /health`)
- ✅ Monitor de heartbeats (detecta workers caídos)
- ✅ Almacenamiento en memoria de workers, jobs y tasks

**Estructuras principales:**
- `Master`: Coordina el sistema completo
- Mapas sincronizados con `sync.RWMutex`
- Monitor de heartbeats en goroutine separada

### 3. Worker (`/worker`)
Nodos de ejecución de tareas:

**Funcionalidades implementadas:**
- ✅ Registro automático con el master al iniciar
- ✅ Envío periódico de heartbeats
- ✅ Gestión de tareas activas
- ✅ Reporte de estado de tareas al master
- ✅ Manejo de señales para shutdown ordenado
- ⏳ Ejecución de operadores (placeholder implementado)

**Estructuras principales:**
- `Worker`: Ejecuta tareas asignadas
- Control de tareas activas con mutex
- Ticker para heartbeats periódicos

## API REST

### Endpoints del Master

#### Registro de Workers
```http
POST /api/v1/workers/register
Content-Type: application/json

{
  "worker_id": "worker-hostname-12345",
  "address": "worker-123:8081"
}
```

#### Heartbeat
```http
POST /api/v1/workers/heartbeat
Content-Type: application/json

{
  "worker_id": "worker-hostname-12345",
  "active_tasks": 2
}
```

#### Crear Job
```http
POST /api/v1/jobs
Content-Type: application/json

{
  "name": "wordcount-batch",
  "dag": {
    "nodes": [
      {"id": "read", "op": "read_csv", "path": "data/*.csv", "partitions": 4}
    ],
    "edges": []
  },
  "parallelism": 4
}
```

#### Listar Jobs
```http
GET /api/v1/jobs
```

#### Obtener Job por ID
```http
GET /api/v1/jobs/{id}
```

#### Health Check
```http
GET /health
```

## Configuración

El sistema se configura mediante variables de entorno:

| Variable | Default | Descripción |
|----------|---------|-------------|
| `MASTER_HOST` | localhost | Host del master |
| `MASTER_PORT` | 8080 | Puerto del master |
| `HEARTBEAT_SEC` | 2 | Intervalo de heartbeats (segundos) |
| `HEARTBEAT_TIMEOUT_SEC` | 10 | Timeout para marcar worker DOWN |
| `WORKER_THREADS` | 4 | Número de threads por worker |
| `MAX_RETRIES` | 3 | Reintentos máximos por tarea |
| `LOG_LEVEL` | INFO | Nivel de logging (DEBUG, INFO, WARN, ERROR) |

## Compilación

```bash
# Compilar master
go build -o bin/master ./master

# Compilar worker
go build -o bin/worker ./worker
```

## Ejecución Local

### Terminal 1 - Master
```bash
export MASTER_HOST=localhost
export MASTER_PORT=8080
export LOG_LEVEL=DEBUG
go run master/main.go
```

### Terminal 2 - Worker 1
```bash
export MASTER_HOST=localhost
export MASTER_PORT=8080
export LOG_LEVEL=DEBUG
go run worker/main.go
```

### Terminal 3 - Worker 2
```bash
export MASTER_HOST=localhost
export MASTER_PORT=8080
export LOG_LEVEL=DEBUG
go run worker/main.go
```

### Probar la API
```bash
# Verificar salud del master
curl http://localhost:8080/health

# Crear un job de prueba
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-job",
    "dag": {"nodes": [], "edges": []},
    "parallelism": 2
  }'

# Listar jobs
curl http://localhost:8080/api/v1/jobs
```

## Próximos Pasos

### Semana 1 ✅
- [x] Estructura de directorios
- [x] Common package con tipos base
- [x] Master básico con registro y heartbeats
- [x] Worker básico con registro y heartbeats
- [x] Comunicación HTTP/JSON funcionando

### Semana 2 (Siguiente)
- [ ] Planificador de tareas (round-robin)
- [ ] Ejecución real de operadores básicos (map, filter)
- [ ] Cliente CLI para enviar jobs
- [ ] Docker Compose para despliegue
- [ ] Métricas básicas de progreso

### Semana 3
- [ ] Operadores avanzados (reduce_by_key, join)
- [ ] Sistema de reintentos
- [ ] Replanificación ante fallos
- [ ] Particionamiento de datos

### Semana 4
- [ ] Observabilidad completa
- [ ] Benchmarks
- [ ] Pruebas automatizadas
- [ ] Video demo

## Principios de Diseño

- **Simplicidad**: Código limpio con nombres significativos (camelCase)
- **Documentación**: Todas las funciones documentadas
- **Concurrencia**: Uso de goroutines y mutexes para sincronización
- **Nivel universitario**: Código educativo y fácil de entender
- **Sin frameworks distribuidos**: Todo implementado desde cero

## Tecnologías

- **Lenguaje**: Go 1.21+
- **Comunicación**: HTTP/1.1 + JSON
- **Concurrencia**: Goroutines + Channels + Mutexes
- **Sin dependencias externas** (solo stdlib)

## Licencia

Proyecto académico - TEC Costa Rica