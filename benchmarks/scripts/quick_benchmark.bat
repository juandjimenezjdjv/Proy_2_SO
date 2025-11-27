@echo off
REM Script de benchmark rápido para Windows
REM Ejecuta benchmarks con dataset pequeño para pruebas

echo ============================================================
echo   BENCHMARK RAPIDO - MINI-SPARK
echo ============================================================
echo.

REM Cambiar al directorio de benchmarks
cd /d "%~dp0.."

echo [INFO] Verificando cluster...
curl -s http://localhost:8080/health > nul 2>&1
if errorlevel 1 (
    echo [ERROR] Cluster no esta activo
    echo [INFO] Inicia el cluster con: docker compose up -d
    pause
    exit /b 1
)
echo [INFO] Cluster activo
echo.

echo [INFO] Generando dataset de prueba (10,000 registros)...
python scripts\generate_benchmark_data.py wordcount data\wordcount_10K.csv 10000
if errorlevel 1 (
    echo [ERROR] Fallo al generar datos
    pause
    exit /b 1
)
echo.

echo [INFO] Para benchmarks completos (1M registros), ejecuta:
echo   python scripts\generate_benchmark_data.py wordcount data\wordcount_1M.csv 1000000
echo   python scripts\generate_benchmark_data.py sales data\sales_1M 1000000 1
echo.

echo [INFO] Los jobs de benchmark estan en: benchmarks\jobs\
echo [INFO] Para enviar un job manualmente:
echo   curl -X POST -H "Content-Type: application/json" -d @benchmarks\jobs\wordcount_benchmark.json http://localhost:8080/api/v1/jobs
echo.

echo [INFO] Consulta benchmarks\README.md para mas informacion
echo.
pause
