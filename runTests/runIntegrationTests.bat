@echo off
setlocal enabledelayedexpansion

REM Script para ejecutar tests de integracion del Mini-Spark Framework
REM Requiere que el cluster este ejecutandose en localhost:8080

echo ==========================================
echo   Mini-Spark Integration Tests Runner
echo ==========================================
echo.

REM Verificar que Go este instalado
where go >nul 2>nul
if !errorlevel! neq 0 (
    echo Error: Go no esta instalado o no esta en el PATH
    pause
    exit /b 1
)

echo Verificando estado del cluster...
curl -s http://localhost:8080/health >nul 2>nul
if !errorlevel! neq 0 (
    echo Error: Cluster no disponible en localhost:8080
    echo    Inicia el cluster con: make docker-up
    pause
    exit /b 1
)

echo Cluster disponible
echo.

echo Ejecutando Tests de Integracion...
echo ==========================================

REM Configurar variables de entorno para tests
set CGO_ENABLED=0

REM Cambiar al directorio padre y ejecutar tests de integracion con verbose output
cd .. && go test ./tests/ -run TestIntegration -v -timeout=5m

REM Capturar el codigo de salida
set TEST_EXIT_CODE=!errorlevel!

echo.
echo ==========================================

if !TEST_EXIT_CODE! equ 0 (
    echo TODOS LOS TESTS DE INTEGRACION PASARON
    echo.
    echo Tests ejecutados:
    echo    * TestIntegrationWordCount
    echo    * TestIntegrationAggregation
    echo    * TestIntegrationPipeline
    echo    * TestIntegrationFailureRecovery
) else (
    echo ALGUNOS TESTS DE INTEGRACION FALLARON
    echo.
)

echo.
echo ==========================================

pause
exit /b !TEST_EXIT_CODE!