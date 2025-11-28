#!/bin/bash

# Script para ejecutar tests de integracion del Mini-Spark Framework
# Requiere que el cluster este ejecutandose en localhost:8080

echo "=========================================="
echo "  Mini-Spark Integration Tests Runner"
echo "=========================================="
echo ""

# Verificar que Go este instalado
if ! command -v go &> /dev/null; then
    echo "Error: Go no esta instalado o no esta en el PATH"
    exit 1
fi

echo "Verificando estado del cluster..."
if ! curl -s http://localhost:8080/health > /dev/null; then
    echo "Error: Cluster no disponible en localhost:8080"
    echo "   Inicia el cluster con: make docker-up"
    exit 1
fi

echo "Cluster disponible"
echo ""

echo "Ejecutando Tests de Integracion..."
echo "=========================================="

# Configurar variables de entorno para tests
export CGO_ENABLED=0

# Cambiar al directorio padre y ejecutar tests de integracion con verbose output
cd .. && go test ./tests/ -run TestIntegration -v -timeout=5m

# Capturar el codigo de salida
TEST_EXIT_CODE=$?

echo ""
echo "=========================================="

if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo "TODOS LOS TESTS DE INTEGRACION PASARON"
    echo ""
    echo "Tests ejecutados:"
    echo "   • TestIntegrationWordCount"
    echo "   • TestIntegrationAggregation" 
    echo "   • TestIntegrationPipeline"
    echo "   • TestIntegrationFailureRecovery"
else
    echo "ALGUNOS TESTS DE INTEGRACION FALLARON"
    echo ""
    echo "REVISAR LA SALIDA ANTERIOR PARA DETALLES"
fi

echo ""
echo "=========================================="

exit $TEST_EXIT_CODE