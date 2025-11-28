@echo off
setlocal enabledelayedexpansion

REM Script para ejecutar tests End-to-End del Mini-Spark Framework
REM Requiere que el cluster Docker este ejecutandose

echo ==========================================
echo    Mini-Spark End-to-End Tests Runner
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
    echo    Inicie el cluster con: make up
    pause
    exit /b 1
)

REM Verificar workers disponibles (simplificado para Windows)
echo Cluster disponible
echo.

echo Ejecutando Tests End-to-End...
echo ==========================================

REM Configurar variables de entorno para tests
set CGO_ENABLED=0

REM Cambiar al directorio padre y ejecutar tests E2E con verbose output y timeout extendido
cd .. && go test ./tests/ -run TestE2E -v -timeout=8m

REM Capturar el codigo de salida
set TEST_EXIT_CODE=!errorlevel!

echo.
echo ==========================================

if !TEST_EXIT_CODE! equ 0 (
    echo TODOS LOS TESTS E2E PASARON
    echo.
    echo Tests ejecutados:
    echo    * TestE2EWordCountMultiNode      - Procesamiento distribuido de texto
    echo    * TestE2EFailureRecoveryMultiNode - Tolerancia a fallos
    echo    * TestE2EComplexPipelineMultiNode - Pipeline con multiples etapas
    echo    * TestE2EConcurrentJobs          - Ejecucion concurrente de trabajos
    echo    * TestE2ELoadBalance            - Distribucion de carga entre workers
    echo.
) else (
    echo ALGUNOS TESTS E2E FALLARON
    echo.
)

echo.
echo ==========================================

pause
exit /b !TEST_EXIT_CODE!