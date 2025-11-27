# 🚀 Mini-Spark - Guía de Ejecución y Uso

## 📋 Tabla de Contenidos
1. [Iniciar el Cluster](#1-iniciar-el-cluster)
2. [Verificar Estado](#2-verificar-estado-del-cluster)
3. [Ver Logs](#3-ver-logs)
4. [Cliente CLI](#4-cliente-cli)
5. [API REST](#5-api-rest-directa)
6. [Detener y Limpiar](#6-detener-y-limpiar)

---

## 1. INICIAR EL CLUSTER

```bash
# Con Docker Compose (Recomendado)
docker compose up -d --build

# Ver estado de los contenedores
docker compose ps
```

**Resultado esperado:**
```
NAME                IMAGE               COMMAND         STATUS
minispark-master    proy_2_so-master    "/app/master"   Up (healthy)
minispark-worker1   proy_2_so-worker1   "/app/worker"   Up
minispark-worker2   proy_2_so-worker2   "/app/worker"   Up
minispark-worker3   proy_2_so-worker3   "/app/worker"   Up
```

---

## 2. VERIFICAR ESTADO DEL CLUSTER

```bash
# Health check con curl
curl http://localhost:8080/health

# O con el cliente CLI
cd client
./client -cmd health
```

**Resultado esperado:**
```json
{
  "master_id": "master-xxxxx",
  "status": "healthy",
  "workers_total": 3,
  "workers_up": 3,
  "timestamp": "2025-11-27T06:15:45Z"
}
```

---

## 3. VER LOGS

```bash
# Logs de TODOS los contenedores
docker compose logs -f

# Solo logs del Master
docker compose logs -f master

# Solo logs de los Workers
docker compose logs -f worker1 worker2 worker3

# Logs de un worker específico
docker compose logs -f worker1

# Últimas 50 líneas
docker compose logs --tail 50
```

**Para salir de los logs:** Presiona `Ctrl + C`

---

## 4. CLIENTE CLI

### Compilar el cliente

```bash
cd client
go build -o client main.go
```

### Comandos disponibles

#### 4.1. Verificar salud del cluster
```bash
./client -cmd health
```

**Salida:**
```
🏥 Estado del Cluster
─────────────────────
  master_id: master-1764224132
  status: healthy
  workers_total: 3
  workers_up: 3
```

#### 4.2. Enviar un job
```bash
./client -cmd submit -job ../data/example-job.json
```

**Salida:**
```
Enviando job 'example-wordcount-job' al Master...
✓ Job enviado exitosamente
  Job ID: example-wordcount-job

Para ver el estado: client -cmd status -id example-wordcount-job
```

#### 4.3. Enviar job con monitoreo en tiempo real
```bash
./client -cmd submit -job ../data/example-job.json -watch
```

**Salida:**
```
Enviando job 'example-wordcount-job' al Master...
✓ Job enviado exitosamente
  Job ID: example-wordcount-job

Monitoreando progreso...
[06:15:45] Estado: RUNNING
  Progreso: 2/8 tareas completadas
[06:15:47] Estado: COMPLETED

✓ Job completado exitosamente
```

#### 4.4. Ver estado de un job específico
```bash
./client -cmd status -id example-wordcount-job
```

**Salida:**
```
📊 Información del Job
━━━━━━━━━━━━━━━━━━━━━━
  ID: example-wordcount-job
  Estado: ACCEPTED
  Nodos: 4
  Enviado: 2025-11-27T06:14:58Z
  
  Tareas (8 total):
    PENDING: 8
```

#### 4.5. Listar todos los jobs
```bash
./client -cmd list
```

**Salida:**
```
Jobs en el sistema: 2

📋 example-wordcount-job
   Estado: COMPLETED
   Nodos: 4
   Enviado: 2025-11-27T06:14:58Z

📋 job-test-001
   Estado: RUNNING
   Nodos: 2
   Enviado: 2025-11-27T06:20:15Z
```

### Opciones del cliente

```bash
./client [opciones]

Opciones:
  -master string
        URL del Master (default "http://localhost:8080")
  -cmd string
        Comando: submit, status, list, health
  -job string
        Archivo JSON con definición del job
  -id string
        ID del job para consultar estado
  -watch
        Monitorear job hasta completarse
```

---

## 5. API REST DIRECTA

### 5.1. Health Check
```bash
curl http://localhost:8080/health
```

### 5.2. Listar jobs
```bash
curl http://localhost:8080/api/v1/jobs
```

### 5.3. Enviar un job
```bash
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "id": "job-test-001",
    "dag": {
      "nodes": [
        {
          "id": "read-data",
          "operator": "read_csv",
          "input_paths": ["input.csv"],
          "output_path": "temp/data.csv",
          "partitions": 3
        },
        {
          "id": "transform",
          "operator": "map",
          "input_paths": ["temp/data.csv"],
          "output_path": "results/output.csv",
          "partitions": 1
        }
      ],
      "edges": [
        {"from": "read-data", "to": "transform"}
      ]
    }
  }'
```

### 5.4. Ver estado de un job
```bash
curl http://localhost:8080/api/v1/jobs/job-test-001
```

---

## 6. DETENER Y LIMPIAR

### Detener el cluster
```bash
docker compose down
```

### Reiniciar el cluster
```bash
docker compose down
docker compose up -d --build
```

### Limpiar todo (imágenes + contenedores)
```bash
docker compose down -v --rmi all
```

### Limpiar datos locales
```bash
rm -rf results/* storage/*
```

---

## 📁 Archivos de Ejemplo

### Estructura de un Job (JSON)

```json
{
  "id": "mi-job-wordcount",
  "dag": {
    "nodes": [
      {
        "id": "read-input",
        "operator": "read_csv",
        "input_paths": ["input.csv"],
        "output_path": "temp/raw.csv",
        "partitions": 3
      },
      {
        "id": "transform",
        "operator": "map",
        "input_paths": ["temp/raw.csv"],
        "output_path": "temp/mapped.csv",
        "partitions": 3
      },
      {
        "id": "count",
        "operator": "reduce_by_key",
        "input_paths": ["temp/mapped.csv"],
        "output_path": "wordcount-result.csv",
        "partitions": 1
      }
    ],
    "edges": [
      {"from": "read-input", "to": "transform"},
      {"from": "transform", "to": "count"}
    ]
  }
}
```

### Datos de entrada (CSV)

Coloca archivos CSV en `./data/`:

```csv
word,category,count
hello,greeting,1
world,noun,1
spark,technology,1
mini,adjective,1
```

### Resultados

Los resultados se guardan en `./results/`:

```bash
ls -la results/
# wordcount-result.csv
# temp/
```

---

## 🎯 Flujo de Trabajo Completo

```bash
# 1. Iniciar cluster
docker compose up -d --build

# 2. Esperar a que esté listo (5 segundos)
sleep 5

# 3. Compilar cliente
cd client && go build -o client main.go

# 4. Verificar salud
./client -cmd health

# 5. Enviar job de ejemplo con monitoreo
./client -cmd submit -job ../data/example-job.json -watch

# 6. Ver resultados
cat ../results/wordcount-result.csv

# 7. Listar todos los jobs
./client -cmd list

# 8. Ver logs si hay problemas
docker compose logs -f

# 9. Detener cuando termines
docker compose down
```

---

## 🔧 Troubleshooting

### Problema: Puerto 8080 ocupado
```bash
# Encuentra qué proceso usa el puerto (Windows)
netstat -ano | findstr :8080

# Detén el proceso o cambia el puerto en docker-compose.yml
```

### Problema: Workers no se registran
```bash
# Ver logs del master
docker compose logs master

# Ver logs de workers
docker compose logs worker1 worker2 worker3

# Reiniciar cluster
docker compose restart
```

### Problema: Contenedores no inician
```bash
# Ver logs de error
docker compose logs

# Reconstruir desde cero
docker compose down
docker compose build --no-cache
docker compose up -d
```

---

## 📊 Comandos Útiles Adicionales

```bash
# Ver uso de recursos
docker stats

# Entrar a un contenedor (debug)
docker exec -it minispark-master sh
docker exec -it minispark-worker1 sh

# Ver redes Docker
docker network ls
docker network inspect proy_2_so_minispark-net

# Ver volúmenes
docker volume ls

# Limpiar todo Docker (¡CUIDADO!)
docker system prune -a
```

---

## 📝 Operadores Disponibles

| Operador | Descripción | Ejemplo |
|----------|-------------|---------|
| `read_csv` | Lee archivos CSV | Leer `input.csv` |
| `map` | Transforma registros 1-a-1 | Convertir a mayúsculas |
| `filter` | Filtra registros | Mantener solo válidos |
| `reduce_by_key` | Agrupa y reduce | Contar por palabra |
| `join` | Une dos datasets | Join por ID |

---

## ✅ Verificación Final

```bash
# 1. Cluster corriendo
docker compose ps
# Deberías ver 4 contenedores: 1 master + 3 workers

# 2. Health check OK
curl http://localhost:8080/health
# Debería mostrar: "status": "healthy", "workers_up": 3

# 3. Cliente funcional
cd client && ./client -cmd health
# Debería mostrar estado del cluster con emojis

# 4. Job de ejemplo ejecutable
./client -cmd submit -job ../data/example-job.json
# Debería responder: "✓ Job enviado exitosamente"
```

**Si todo lo anterior funciona, ¡el sistema está completamente operativo!** 🎉
