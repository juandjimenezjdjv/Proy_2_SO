# 🚀 Mini-Spark - Guía de Uso Rápida

## Iniciar el Cluster

```bash
# Opción 1: Con Docker Compose
docker compose up -d --build

# Verificar que todo está corriendo
docker compose ps
curl http://localhost:8080/health
```

## Cliente CLI

### Compilar el cliente localmente
```bash
cd client
go build -o client main.go
```

### Comandos disponibles

#### 1. Verificar salud del cluster
```bash
./client -cmd health
```

#### 2. Enviar un job
```bash
./client -cmd submit -job ../data/example-job.json
```

#### 3. Enviar y monitorear en tiempo real
```bash
./client -cmd submit -job ../data/example-job.json -watch
```

#### 4. Ver estado de un job
```bash
./client -cmd status -id example-wordcount-job
```

#### 5. Listar todos los jobs
```bash
./client -cmd list
```

## Estructura de un Job

Un job se define en JSON con la siguiente estructura:

```json
{
  "id": "mi-job-001",
  "dag": {
    "nodes": [
      {
        "id": "leer-datos",
        "operator": "read_csv",
        "input_paths": ["input.csv"],
        "output_path": "temp/datos-leidos.csv",
        "partitions": 3
      },
      {
        "id": "transformar",
        "operator": "map",
        "input_paths": ["temp/datos-leidos.csv"],
        "output_path": "resultado.csv",
        "partitions": 1
      }
    ],
    "edges": [
      {"from": "leer-datos", "to": "transformar"}
    ]
  }
}
```

## Operadores Disponibles

- **read_csv**: Lee archivos CSV
- **map**: Transforma cada registro (1-a-1)
- **filter**: Filtra registros según condición
- **reduce_by_key**: Agrupa y reduce por clave
- **join**: Une dos datasets por clave

## Ver Logs

```bash
# Todos los logs
docker compose logs -f

# Solo master
docker compose logs -f master

# Solo workers
docker compose logs -f worker1 worker2 worker3
```

## Detener el Cluster

```bash
docker compose down
```

## Ejemplo Completo

```bash
# 1. Iniciar cluster
docker compose up -d --build

# 2. Esperar a que esté listo (5 segundos)
sleep 5

# 3. Compilar cliente
cd client && go build -o client main.go

# 4. Enviar job de ejemplo
./client -cmd submit -job ../data/example-job.json -watch

# 5. Ver resultados
cat ../results/wordcount-result.csv
```

## Archivos de Datos

- **Entrada**: Coloca archivos CSV en `./data/`
- **Resultados**: Se guardan en `./results/`
- **Temporales**: Se crean en `./storage/` (automático)
