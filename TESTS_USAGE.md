# 🧪 Scripts de Tests Unitarios

Este directorio contiene scripts para ejecutar fácilmente todos los tests unitarios del proyecto Mini-Spark.

## 📋 Archivos Disponibles

- **`run_tests.bat`** - Script para Windows (CMD/PowerShell)
- **`run_tests.sh`** - Script para Linux/Mac (Bash)

## 🚀 Uso

### Windows

```cmd
.\run_tests.bat
```

O simplemente hacer doble clic en el archivo `run_tests.bat` desde el explorador de archivos.

### Linux/Mac

```bash
chmod +x run_tests.sh  # Solo la primera vez
./run_tests.sh
```

## 📊 Lo que hace el script

1. Verifica que estés en el directorio correcto (busca `go.mod`)
2. Ejecuta todos los tests con `go test ./... -v -race`
3. Muestra el output completo de todos los tests
4. Al finalizar, muestra un resumen con:
   - ✅ **76 tests** distribuidos en 4 archivos
   - 📦 `common/types_test.go`: 29 tests
   - 📦 `common/cache_test.go`: 15 tests
   - 📦 `master/scheduler_test.go`: 15 tests
   - 📦 `worker/executor_test.go`: 17 tests

## 🎯 Resultado Esperado

Si todos los tests pasan, verás:

```
╔════════════════════════════════════════════════╗
║   TODOS LOS TESTS PASARON ✓                    ║
╚════════════════════════════════════════════════╝

📊 TOTAL: 76 tests (100% passing)
```

Si algún test falla, verás:

```
╔════════════════════════════════════════════════╗
║   TESTS FALLIDOS                               ║
╚════════════════════════════════════════════════╝

[ERROR] Algunos tests fallaron. Revisa el output arriba.
```

## 🔧 Opciones Avanzadas

Para ejecutar tests manualmente con más control:

```bash
# Ejecutar solo tests de un paquete
go test ./common -v

# Ejecutar con cobertura
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Ejecutar sin cache (forzar re-ejecución)
go test ./... -count=1

# Ejecutar tests específicos
go test ./worker -run TestMap -v
```

## 📝 Notas

- Los tests usan el **race detector** (`-race`) para detectar condiciones de carrera
- Cada test se ejecuta en un directorio temporal aislado (`t.TempDir()`)
- Los tests no requieren que el servidor esté corriendo
- Tiempo de ejecución total: ~4-5 segundos

## ✅ Estado Actual

**76/76 tests pasando (100%)**

- ✅ Estructuras básicas (Job, Task, DAG)
- ✅ Sistema de cache con spill-to-disk
- ✅ Planificación DAG (algoritmo de Kahn)
- ✅ Todos los operadores (read_csv, map, filter, flat_map, reduce_by_key, aggregate, join)
- ✅ Tolerancia a fallos (reintentos, reasignación)
- ✅ Particionamiento hash-based
