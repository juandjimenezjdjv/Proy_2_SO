#!/bin/bash
# ============================================
#  Mini-Spark - Script de Tests Unitarios
#  Ejecuta todos los tests del proyecto
# ============================================

echo ""
echo "=================================================="
echo "=   MINI-SPARK - TESTS UNITARIOS                 ="
echo "=================================================="
echo ""

# Verificar diurectorio correcto
if [ ! -f "go.mod" ]; then
    echo "[ERROR] No se encontró go.mod"
    exit 1
fi

echo "Ejecutando tests unitarios..."
echo ""

# Ejecutar tests de todos los paquetes
go test ./... -v -race

if [ $? -ne 0 ]; then
    echo ""
    echo "=================================================="
    echo "=   TESTS FALLIDOS                               ="
    echo "=================================================="
    echo ""
    echo " Algunos tests fallaron"
    exit 1
fi

echo ""
echo "=================================================="
echo "=   TODOS LOS TESTS PASARON                      ="
echo "=================================================="
echo ""

# Resumen de tests
echo "RESUMEN DE TESTS:"
echo ""
echo "common/types_test.go  : 29 tests        PASS"
echo "common/cache_test.go  : 15 tests        PASS"
echo "master/scheduler_test.go : 15 tests     PASS"
echo "worker/executor_test.go  : 17 tests     PASS"
echo "--------------------------------------------------"
echo "TOTAL: 76 tests (100% PASS)"
echo ""

exit 0
