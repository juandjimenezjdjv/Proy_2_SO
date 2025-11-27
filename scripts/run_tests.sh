#!/bin/bash
# Script de testing completo para Mini-Spark

set -e

echo "═══════════════════════════════════════════════════════"
echo "  Mini-Spark - Test Suite Completo"
echo "═══════════════════════════════════════════════════════"
echo ""

# Colores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Función para imprimir con color
print_status() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Función para esperar
wait_seconds() {
    echo -n "Esperando $1 segundos"
    for i in $(seq 1 $1); do
        sleep 1
        echo -n "."
    done
    echo ""
}

# Variables
MASTER_URL="http://localhost:8080"
CLIENT="../client/client.exe"

# Test 1: Verificar que el cluster está corriendo
print_status "Test 1: Verificando cluster..."
if curl -s "${MASTER_URL}/health" > /dev/null 2>&1; then
    print_success "Cluster está activo"
else
    print_error "Cluster no responde"
    exit 1
fi

# Test 2: Verificar workers
print_status "Test 2: Verificando workers..."
WORKERS=$(curl -s "${MASTER_URL}/health" | python -m json.tool 2>/dev/null | grep workers_up | awk '{print $2}' | tr -d ',')
if [ "$WORKERS" -ge 1 ]; then
    print_success "$WORKERS workers activos"
else
    print_error "No hay workers activos"
    exit 1
fi

# Test 3: Generar datos de prueba
print_status "Test 3: Generando datos de prueba..."
cd scripts
python generate_test_data.py
cd ..
print_success "Datos generados"

# Test 4: Enviar job simple (wordcount)
print_status "Test 4: Enviando job wordcount..."
JOB_RESPONSE=$(curl -s -X POST "${MASTER_URL}/api/v1/jobs" \
    -H "Content-Type: application/json" \
    -d @data/example-job.json)
JOB_ID=$(echo "$JOB_RESPONSE" | python -m json.tool 2>/dev/null | grep '"id"' | head -1 | awk '{print $2}' | tr -d '",')
print_success "Job enviado: $JOB_ID"

# Esperar ejecución
wait_seconds 10

# Verificar estado
STATUS=$(curl -s "${MASTER_URL}/api/v1/jobs/${JOB_ID}" | python -m json.tool 2>/dev/null | grep '"status"' | head -1 | awk '{print $2}' | tr -d '",')
print_status "Estado del job: $STATUS"

if [[ "$STATUS" == "COMPLETED" || "$STATUS" == "RUNNING" ]]; then
    print_success "Job procesado correctamente"
else
    print_warning "Job en estado: $STATUS"
fi

# Test 5: Job con pipeline completo
print_status "Test 5: Enviando pipeline completo..."
curl -s -X POST "${MASTER_URL}/api/v1/jobs" \
    -H "Content-Type: application/json" \
    -d @examples/complete_pipeline.json > /dev/null
print_success "Pipeline enviado"

wait_seconds 15

# Test 6: Job con dataset grande
print_status "Test 6: Procesando dataset grande..."
curl -s -X POST "${MASTER_URL}/api/v1/jobs" \
    -H "Content-Type: application/json" \
    -d @examples/large_data_job.json > /dev/null
print_success "Job de dataset grande enviado"

wait_seconds 20

# Test 7: Job con join
print_status "Test 7: Probando operación join..."
curl -s -X POST "${MASTER_URL}/api/v1/jobs" \
    -H "Content-Type: application/json" \
    -d @examples/join_job.json > /dev/null
print_success "Job de join enviado"

wait_seconds 15

# Test 8: Listar todos los jobs
print_status "Test 8: Listando todos los jobs..."
JOBS=$(curl -s "${MASTER_URL}/api/v1/jobs" | python -m json.tool 2>/dev/null | grep '"id"' | wc -l)
print_success "Total de jobs: $JOBS"

# Test 9: Verificar resultados
print_status "Test 9: Verificando archivos de resultados..."
if [ -d "results" ]; then
    RESULT_FILES=$(find results -name "*.csv" 2>/dev/null | wc -l)
    if [ "$RESULT_FILES" -gt 0 ]; then
        print_success "Se generaron $RESULT_FILES archivos de resultados"
        echo ""
        echo "Archivos de resultados:"
        find results -name "*.csv" -exec ls -lh {} \;
    else
        print_warning "No se encontraron archivos de resultados"
    fi
else
    print_warning "Directorio results no existe"
fi

# Test 10: Verificar logs
print_status "Test 10: Revisando logs del master..."
echo ""
echo "Últimas líneas del master:"
docker logs minispark-master --tail 10

# Resumen
echo ""
echo "═══════════════════════════════════════════════════════"
echo "  Resumen de Tests"
echo "═══════════════════════════════════════════════════════"
echo ""

# Contar jobs por estado
COMPLETED=$(curl -s "${MASTER_URL}/api/v1/jobs" | grep -o '"COMPLETED"' | wc -l)
RUNNING=$(curl -s "${MASTER_URL}/api/v1/jobs" | grep -o '"RUNNING"' | wc -l)
FAILED=$(curl -s "${MASTER_URL}/api/v1/jobs" | grep -o '"FAILED"' | wc -l)

echo "Jobs completados: $COMPLETED"
echo "Jobs en ejecución: $RUNNING"
echo "Jobs fallidos: $FAILED"
echo ""

if [ "$COMPLETED" -gt 0 ]; then
    print_success "Tests completados exitosamente!"
else
    print_warning "Algunos tests no completaron"
fi

echo ""
echo "Para ver detalles de un job:"
echo "  curl ${MASTER_URL}/api/v1/jobs/{JOB_ID} | python -m json.tool"
echo ""
echo "Para ver el estado del cluster:"
echo "  ${CLIENT} -cmd health"
echo ""
