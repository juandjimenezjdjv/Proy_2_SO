#!/bin/bash

# Script para ejecutar tests End-to-End del Mini-Spark Framework
# Requiere que el cluster Docker esté ejecutándose

echo "=========================================="
echo "   Mini-Spark End-to-End Tests Runner"
echo "=========================================="
echo ""

# Verificar que Go esté instalado
if ! command -v go &> /dev/null; then
    echo "Error: Go no está instalado o no está en el PATH"
    exit 1
fi

echo "Verificando estado del cluster..."
if ! curl -s http://localhost:8080/health > /dev/null; then
    echo "Error: Cluster no disponible en localhost:8080"
    echo "Inicie el cluster con: make up"
    exit 1
fi

# Verificar workers disponibles
WORKERS_UP=$(curl -s http://localhost:8080/health | grep -o '"workers_up":[0-9]*' | cut -d':' -f2)
if [ "$WORKERS_UP" -lt 3 ]; then
    echo "[WARNING] Solo $WORKERS_UP workers disponibles (recomendado: 3)"
    echo "Los tests pueden ser lentos o fallar"
else
    echo "Cluster con $WORKERS_UP workers disponible"
fi

echo ""

echo "Ejecutando Tests End-to-End..."
echo "========================================="

# Configurar variables de entorno para tests
export CGO_ENABLED=0

# Cambiar al directorio padre y ejecutar tests E2E con verbose output y timeout extendido
cd .. && go test ./tests/ -run TestE2E -v -timeout=8m

# Capturar el código de salida
TEST_EXIT_CODE=$?

echo ""
echo "=========================================="

if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo "TODOS LOS TESTS E2E PASARON"
    echo ""
    echo "Tests ejecutados:"
    echo "   • TestE2EWordCountMultiNode      - Procesamiento distribuido de texto"
    echo "   • TestE2EFailureRecoveryMultiNode - Tolerancia a fallos"
    echo "   • TestE2EComplexPipelineMultiNode - Pipeline con múltiples etapas"
    echo "   • TestE2EConcurrentJobs          - Ejecución concurrente de trabajos"
    echo "   • TestE2ELoadBalance            - Distribución de carga entre workers"
    echo ""
else
    echo "ALGUNOS TESTS E2E FALLARON"
    echo ""
    echo "Revisar logs para más detalles"
fi

echo ""
echo "=========================================="

exit $TEST_EXIT_CODE