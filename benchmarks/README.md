# Benchmarks - Mini-Spark

Esta carpeta contiene todos los recursos necesarios para ejecutar benchmarks exhaustivos del sistema Mini-Spark.

## 📁 Estructura

```
benchmarks/
├── scripts/
│   ├── generate_benchmark_data.py  # Generador de datasets
│   └── benchmark.sh                # Suite de benchmarks automatizada
├── data/                           # Datasets generados (gitignored)
│   ├── wordcount_1M.csv
│   └── sales_1M.csv
├── jobs/                           # Definiciones de jobs
│   ├── wordcount_benchmark.json
│   ├── aggregate_benchmark.json
│   ├── filter_benchmark.json
│   └── complex_pipeline_benchmark.json
├── results/                        # Resultados de ejecución (gitignored)
├── BENCHMARKS.md                   # Documentación y resultados
└── README.md                       # Este archivo
```

## 🚀 Inicio Rápido

### 1. Preparar el Cluster

```bash
# Desde la raíz del proyecto
docker compose up -d --build

# Verificar que esté activo
curl http://localhost:8080/health
```

### 2. Ejecutar Benchmarks

```bash
# Ejecutar suite completa
cd benchmarks
bash scripts/benchmark.sh
```

El script automáticamente:
- ✅ Verifica que el cluster esté activo
- ✅ Genera datasets de 1M registros
- ✅ Ejecuta 4 benchmarks diferentes
- ✅ Mide tiempo, throughput y recursos
- ✅ Genera reporte en `BENCHMARK_REPORT.txt`

### 3. Ver Resultados

```bash
# Reporte de texto
cat BENCHMARK_REPORT.txt

# Documentación completa
cat BENCHMARKS.md

# Archivos CSV generados
ls -lh results/
```

## 📊 Benchmarks Incluidos

### 1. WordCount (flat_map intensivo)
- **Dataset:** 1M líneas de texto
- **Operaciones:** read_csv → flat_map (tokenize) → map (lowercase) → reduce_by_key
- **Mide:** Performance de flat_map y procesamiento de texto

### 2. Aggregate (reduce intensivo)
- **Dataset:** 1M registros de ventas
- **Operaciones:** read_csv → filter (status) → reduce_by_key (sum by city)
- **Mide:** Agregación y filtrado

### 3. Filter (selectividad)
- **Dataset:** 1M registros de ventas
- **Operaciones:** read_csv → filter (price > 500) → filter (category) → write
- **Mide:** Filtros encadenados y selectividad

### 4. Complex Pipeline
- **Dataset:** 1M registros de ventas
- **Operaciones:** read → map (revenue) → filter → reduce (x2 paralelo)
- **Mide:** DAGs complejos con bifurcaciones

## 🛠️ Generación Manual de Datos

### WordCount Dataset

```bash
python3 scripts/generate_benchmark_data.py wordcount data/wordcount_1M.csv 1000000
```

Genera 1M líneas con palabras técnicas (spark, hadoop, kafka, etc.)

### Sales Dataset

```bash
python3 scripts/generate_benchmark_data.py sales data/sales_1M 1000000 1
```

Genera 1M registros de ventas con: id, timestamp, product, category, quantity, price, city, status

### Datasets Particionados

```bash
# Generar con 4 particiones
python3 scripts/generate_benchmark_data.py sales data/sales_1M 1000000 4

# Genera: sales_1M-part-0.csv, sales_1M-part-1.csv, etc.
```

## 📝 Ejecución Manual de Jobs

Si prefieres ejecutar jobs individuales:

```bash
# 1. Generar datos
python3 scripts/generate_benchmark_data.py wordcount data/wordcount_1M.csv 1000000

# 2. Enviar job
curl -X POST -H "Content-Type: application/json" \
  -d @jobs/wordcount_benchmark.json \
  http://localhost:8080/api/v1/jobs

# 3. Obtener job_id y monitorear
curl http://localhost:8080/api/v1/jobs/<job_id>
```

## 📈 Métricas Recolectadas

El sistema mide:

- ⏱️ **Tiempo de ejecución total** (inicio → completado)
- 🚀 **Throughput** (registros/segundo)
- 💾 **Uso de memoria** (peak y promedio)
- ⚙️ **CPU utilization** por worker
- 📊 **Progreso en tiempo real** (%)
- 🔄 **Cache hits/misses y spills**

## 🎯 Requisitos del Proyecto

Estos benchmarks cumplen con:

> **PDF Sección 11:** "Benchmarks mínimos: lote de 1M registros (Batch)"  
> **PDF Sección 15 Rúbrica:** "Demo y benchmarks" - 5%

Características:
- ✅ 1 millón de registros procesados
- ✅ Múltiples tipos de operaciones (map, filter, flat_map, reduce)
- ✅ Medición de throughput y latencia
- ✅ Diferentes configuraciones (workers, particiones)
- ✅ Reporte documentado

## 🔧 Configuración Avanzada

### Modificar Número de Particiones

Edita el archivo JSON del job:

```json
{
  "stage_id": "read",
  "operator": "read_csv",
  "config": {
    "input_path": "./benchmarks/data/sales_1M.csv",
    "partitions": 8  // Cambiar aquí
  }
}
```

### Escalar Workers

```bash
# En docker-compose.yml, agregar más workers:
docker compose up -d --scale worker=5
```

### Ajustar Límite de Memoria

```bash
# En docker-compose.yml:
environment:
  - MAX_MEMORY_MB=200  # Cambiar de 100 a 200
```

## 🐛 Troubleshooting

### Error: "Cluster no está activo"

```bash
docker compose up -d
# Esperar 5-10 segundos para que inicien
curl http://localhost:8080/health
```

### Error: "python3: command not found"

```bash
# Instalar Python 3
# Windows: https://www.python.org/downloads/
# Linux: sudo apt install python3
```

### Benchmark se queda en "PENDING"

```bash
# Verificar workers
curl http://localhost:8080/api/v1/workers

# Ver logs
docker logs minispark-master
docker logs minispark-worker1
```

### Dataset muy grande (disco lleno)

```bash
# Reducir tamaño
python3 scripts/generate_benchmark_data.py sales data/sales_100K 100000 1

# Limpiar datos antiguos
rm -rf data/*.csv
```

## 📚 Referencias

- [BENCHMARKS.md](BENCHMARKS.md) - Documentación completa y resultados
- [../README.md](../README.md) - Documentación principal del proyecto
- [../TODO.md](../TODO.md) - Estado de implementación

---

**Nota:** Los archivos `data/` y `results/` están en `.gitignore` porque pueden ser muy grandes. Se generan automáticamente al ejecutar los benchmarks.

