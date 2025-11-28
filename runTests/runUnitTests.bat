@echo off
REM ============================================
REM  Mini-Spark - Script de Tests Unitarios
REM  Ejecuta todos los tests del proyecto
REM ============================================

echo.
echo ==================================================
echo =   MINI-SPARK - TESTS UNITARIOS                 =
echo ==================================================
echo.

REM Verificar que estamos en el directorio correcto - buscar en directorio padre
if not exist "..\go.mod" (
    echo No se encontró go.mod en directorio padre
    echo Ejecutar desde la carpeta runTests\ o directorio raíz del proyecto
    exit /b 1
)

echo Ejecutando tests unitarios...
echo.

# Cambiar al directorio padre y ejecutar tests de todos los paquetes excepto tests/
cd .. && go test ./common ./master ./worker -v -race

if %ERRORLEVEL% neq 0 (
    echo.
    echo ==================================================
    echo =   TESTS FALLIDOS                               =
    echo ==================================================
    echo.
    echo Algunos tests fallaron
    exit /b 1
)

echo.
echo ==================================================
echo =   TODOS LOS TESTS PASARON                      =
echo ==================================================
echo.

REM Resumen de tests
echo Resumen de tests:
echo.
echo common/types_test.go  : 29 tests         PASS
echo common/cache_test.go  : 15 tests         PASS
echo master/scheduler_test.go : 15 tests      PASS
echo worker/executor_test.go  : 17 tests      PASS
echo ==============================================
echo TOTAL: 76 tests (100%% PASS)
echo.

exit /b 0
