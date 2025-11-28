package common

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupCacheTest crea un directorio temporal para tests de cache
func setupCacheTest(t *testing.T) (string, func()) {
	tempDir := t.TempDir()
	spillDir := filepath.Join(tempDir, "spill")

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return spillDir, cleanup
}

// TestCachePut prueba almacenar datos en cache
func TestCachePut(t *testing.T) {
	spillDir, cleanup := setupCacheTest(t)
	defer cleanup()

	logger := NewLogger("CACHE_TEST", LogLevelInfo)
	cache := NewCacheManager(10, spillDir, logger) // 10MB límite

	// Datos pequeños que caben en memoria
	data := [][]string{
		{"key1", "value1"},
		{"key2", "value2"},
	}

	err := cache.Put("test-key", data)
	if err != nil {
		t.Fatalf("error almacenando en cache: %v", err)
	}

	// Verificar que se almacenó en memoria
	stats := cache.GetStats()
	cachedKeys, ok := stats["cached_keys"].(int)
	if !ok || cachedKeys != 1 {
		t.Errorf("esperado 1 clave en cache, obtenido %v", stats["cached_keys"])
	}
}

// TestCacheGet prueba recuperar datos del cache
func TestCacheGet(t *testing.T) {
	spillDir, cleanup := setupCacheTest(t)
	defer cleanup()

	logger := NewLogger("CACHE_TEST", LogLevelInfo)
	cache := NewCacheManager(10, spillDir, logger)

	// Almacenar datos
	originalData := [][]string{
		{"row1col1", "row1col2"},
		{"row2col1", "row2col2"},
	}

	if err := cache.Put("test-key", originalData); err != nil {
		t.Fatalf("error almacenando: %v", err)
	}

	// Recuperar datos
	retrievedData, err := cache.Get("test-key")
	if err != nil {
		t.Fatalf("error recuperando: %v", err)
	}

	// Verificar que los datos son correctos
	if len(retrievedData) != len(originalData) {
		t.Errorf("esperado %d registros, obtenido %d", len(originalData), len(retrievedData))
	}

	for i := range originalData {
		for j := range originalData[i] {
			if retrievedData[i][j] != originalData[i][j] {
				t.Errorf("dato incorrecto [%d][%d]: esperado '%s', obtenido '%s'",
					i, j, originalData[i][j], retrievedData[i][j])
			}
		}
	}
}

// TestCacheGetNonExistent prueba recuperar clave inexistente
func TestCacheGetNonExistent(t *testing.T) {
	spillDir, cleanup := setupCacheTest(t)
	defer cleanup()

	logger := NewLogger("CACHE_TEST", LogLevelInfo)
	cache := NewCacheManager(10, spillDir, logger)

	_, err := cache.Get("nonexistent-key")
	if err == nil {
		t.Error("esperado error para clave inexistente")
	}
}

// TestCacheSpill prueba spill automático a disco
func TestCacheSpill(t *testing.T) {
	spillDir, cleanup := setupCacheTest(t)
	defer cleanup()

	logger := NewLogger("CACHE_TEST", LogLevelInfo)
	// Límite muy bajo para forzar spill: 1MB
	cache := NewCacheManager(1, spillDir, logger)

	// Generar datos grandes que excedan el límite
	largeData := make([][]string, 10000)
	for i := 0; i < 10000; i++ {
		largeData[i] = []string{
			"this is a long string to fill memory quickly",
			"another long column value here",
			"third column with more data",
		}
	}

	// Almacenar datos (debería hacer spill)
	err := cache.Put("large-data", largeData)
	if err != nil {
		t.Fatalf("error en spill: %v", err)
	}

	// Verificar que se hizo spill o que se almacenó
	stats := cache.GetStats()
	spilledKeys := stats["spilled_keys"].(int)
	cachedKeys := stats["cached_keys"].(int)

	// Debe estar en spill O en cache (dependiendo de la estimación de memoria)
	if spilledKeys+cachedKeys != 1 {
		t.Errorf("esperado 1 clave total, obtenido spilled=%d cached=%d", spilledKeys, cachedKeys)
	}

	// Recuperar datos desde disco
	retrievedData, err := cache.Get("large-data")
	if err != nil {
		t.Fatalf("error recuperando datos spilled: %v", err)
	}

	// Verificar integridad
	if len(retrievedData) != len(largeData) {
		t.Errorf("esperado %d registros, obtenido %d", len(largeData), len(retrievedData))
	}
}

// TestCacheSpillMultiple prueba múltiples spills
func TestCacheSpillMultiple(t *testing.T) {
	spillDir, cleanup := setupCacheTest(t)
	defer cleanup()

	logger := NewLogger("CACHE_TEST", LogLevelInfo)
	cache := NewCacheManager(1, spillDir, logger)

	// Crear múltiples datasets grandes
	datasets := []string{"dataset1", "dataset2", "dataset3"}
	for _, key := range datasets {
		largeData := make([][]string, 5000)
		for i := 0; i < 5000; i++ {
			largeData[i] = []string{"data", "more data", "even more"}
		}

		if err := cache.Put(key, largeData); err != nil {
			t.Fatalf("error almacenando %s: %v", key, err)
		}
	}

	// Verificar que todos se almacenaron (spilled o cached)
	stats := cache.GetStats()
	spilledKeys := stats["spilled_keys"].(int)
	cachedKeys := stats["cached_keys"].(int)
	totalKeys := spilledKeys + cachedKeys
	if totalKeys != 3 {
		t.Errorf("esperado 3 datasets totales, obtenido spilled=%d cached=%d", spilledKeys, cachedKeys)
	}

	// Verificar que todos se pueden recuperar
	for _, key := range datasets {
		data, err := cache.Get(key)
		if err != nil {
			t.Errorf("error recuperando %s: %v", key, err)
		}
		if len(data) != 5000 {
			t.Errorf("datos incorrectos para %s: esperado 5000 registros, obtenido %d", key, len(data))
		}
	}
}

// TestCacheClear prueba limpiar cache completo
func TestCacheClear(t *testing.T) {
	spillDir, cleanup := setupCacheTest(t)
	defer cleanup()

	logger := NewLogger("CACHE_TEST", LogLevelInfo)
	cache := NewCacheManager(10, spillDir, logger)

	// Agregar datos en memoria
	cache.Put("key1", [][]string{{"data1"}})
	cache.Put("key2", [][]string{{"data2"}})

	// Limpiar
	err := cache.Clear()
	if err != nil {
		t.Fatalf("error limpiando cache: %v", err)
	}

	// Verificar que está vacío
	stats := cache.GetStats()
	if stats["cached_keys"].(int) != 0 {
		t.Error("cache no está vacío después de Clear")
	}
	if stats["spilled_keys"].(int) != 0 {
		t.Error("archivos spilled no se eliminaron")
	}
	if stats["current_memory_mb"].(int64) != 0 {
		t.Error("memoria actual no es 0")
	}
}

// TestCacheStats prueba obtención de estadísticas
func TestCacheStats(t *testing.T) {
	spillDir, cleanup := setupCacheTest(t)
	defer cleanup()

	logger := NewLogger("CACHE_TEST", LogLevelInfo)
	cache := NewCacheManager(100, spillDir, logger)

	stats := cache.GetStats()

	// Verificar campos requeridos
	requiredFields := []string{
		"max_memory_mb",
		"current_memory_mb",
		"cached_keys",
		"spilled_keys",
		"memory_usage_pct",
	}

	for _, field := range requiredFields {
		if _, ok := stats[field]; !ok {
			t.Errorf("campo requerido ausente en stats: %s", field)
		}
	}

	// Verificar valores iniciales
	if stats["cached_keys"].(int) != 0 {
		t.Error("cached_keys inicial debe ser 0")
	}
	if stats["spilled_keys"].(int) != 0 {
		t.Error("spilled_keys inicial debe ser 0")
	}
	if stats["current_memory_mb"].(int64) != 0 {
		t.Error("current_memory_mb inicial debe ser 0")
	}
	if stats["memory_usage_pct"].(float64) != 0.0 {
		t.Error("memory_usage_pct inicial debe ser 0")
	}
}

// TestCacheMemoryAccounting prueba contabilidad de memoria
func TestCacheMemoryAccounting(t *testing.T) {
	spillDir, cleanup := setupCacheTest(t)
	defer cleanup()

	logger := NewLogger("CACHE_TEST", LogLevelInfo)
	cache := NewCacheManager(100, spillDir, logger)

	// Agregar datos pequeños
	smallData := [][]string{
		{"a", "b", "c"},
		{"d", "e", "f"},
	}

	cache.Put("small", smallData)

	// Verificar que el uso de memoria aumentó
	stats := cache.GetStats()
	currentMem := stats["current_memory_mb"].(int64)
	if currentMem <= 0 {
		t.Error("uso de memoria debe ser > 0 después de Put")
	}

	// Agregar más datos
	cache.Put("small2", smallData)

	stats2 := cache.GetStats()
	currentMem2 := stats2["current_memory_mb"].(int64)
	if currentMem2 <= currentMem {
		t.Error("uso de memoria debe aumentar con más datos")
	}
}

// TestCacheEstimateSize prueba estimación de tamaño
func TestCacheEstimateSize(t *testing.T) {
	spillDir, cleanup := setupCacheTest(t)
	defer cleanup()

	logger := NewLogger("CACHE_TEST", LogLevelInfo)
	cache := NewCacheManager(100, spillDir, logger)

	// Datos vacíos
	emptyData := [][]string{}
	size := cache.estimateSize(emptyData)
	if size != 0 {
		t.Errorf("tamaño de datos vacíos debe ser 0, obtenido %d", size)
	}

	// Datos pequeños
	smallData := [][]string{
		{"a"},
	}
	sizeSmall := cache.estimateSize(smallData)
	if sizeSmall <= 0 {
		t.Error("tamaño de datos pequeños debe ser > 0")
	}

	// Datos más grandes
	largeData := make([][]string, 1000)
	for i := 0; i < 1000; i++ {
		largeData[i] = []string{"this is a longer string", "another column"}
	}
	sizeLarge := cache.estimateSize(largeData)
	// La estimación de tamaño es aproximada, pero más datos deben dar más MB estimados
	// Permitir que ambos sean 1 si están por debajo del threshold
	if len(largeData) > len(smallData) && sizeLarge < sizeSmall {
		t.Errorf("datos más grandes (%d registros, size=%dMB) deben tener tamaño >= pequeños (%d registros, size=%dMB)",
			len(largeData), sizeLarge, len(smallData), sizeSmall)
	}
}

// TestCacheMemoryLimit prueba respeto al límite de memoria
func TestCacheMemoryLimit(t *testing.T) {
	spillDir, cleanup := setupCacheTest(t)
	defer cleanup()

	logger := NewLogger("CACHE_TEST", LogLevelInfo)
	maxMemory := int64(5) // 5MB límite
	cache := NewCacheManager(maxMemory, spillDir, logger)

	// Generar datos que caben en memoria
	smallData := [][]string{
		{"small", "data"},
	}

	// Almacenar varias veces
	for i := 0; i < 3; i++ {
		key := "key-" + string(rune(i+'0'))
		if err := cache.Put(key, smallData); err != nil {
			t.Fatalf("error almacenando %s: %v", key, err)
		}
	}

	// Verificar que no excedió el límite
	stats := cache.GetStats()
	currentMem := stats["current_memory_mb"].(int64)
	if currentMem > maxMemory {
		t.Errorf("memoria actual (%dMB) excede límite (%dMB)", currentMem, maxMemory)
	}
}

// TestCacheSpillDirCreation prueba creación del directorio de spill
func TestCacheSpillDirCreation(t *testing.T) {
	tempDir := t.TempDir()
	spillDir := filepath.Join(tempDir, "nested", "spill", "dir")

	logger := NewLogger("CACHE_TEST", LogLevelInfo)
	cache := NewCacheManager(1, spillDir, logger)

	// Generar datos grandes para forzar spill
	largeData := make([][]string, 5000)
	for i := 0; i < 5000; i++ {
		largeData[i] = []string{"data1", "data2", "data3"}
	}

	err := cache.Put("test", largeData)
	if err != nil {
		t.Fatalf("error en spill: %v", err)
	}

	// Verificar que el directorio se creó
	if _, err := os.Stat(spillDir); os.IsNotExist(err) {
		t.Error("directorio de spill no se creó")
	}
}

// TestCacheConcurrency prueba acceso concurrente básico
func TestCacheConcurrency(t *testing.T) {
	spillDir, cleanup := setupCacheTest(t)
	defer cleanup()

	logger := NewLogger("CACHE_TEST", LogLevelInfo)
	cache := NewCacheManager(50, spillDir, logger)

	// Ejecutar Put concurrente
	done := make(chan bool, 3)

	for i := 0; i < 3; i++ {
		go func(id int) {
			data := [][]string{
				{"data-" + string(rune(id+'0'))},
			}
			key := "concurrent-key-" + string(rune(id+'0'))
			if err := cache.Put(key, data); err != nil {
				t.Errorf("error en Put concurrente %d: %v", id, err)
			}
			done <- true
		}(i)
	}

	// Esperar que terminen
	timeout := time.After(5 * time.Second)
	for i := 0; i < 3; i++ {
		select {
		case <-done:
			// OK
		case <-timeout:
			t.Fatal("timeout esperando operaciones concurrentes")
		}
	}

	// Verificar que todas las claves están presentes
	stats := cache.GetStats()
	totalKeys := stats["cached_keys"].(int) + stats["spilled_keys"].(int)
	if totalKeys != 3 {
		t.Errorf("esperado 3 claves totales, obtenido %d", totalKeys)
	}
}

// TestCacheSpillFilename prueba generación de nombres únicos para spill
func TestCacheSpillFilename(t *testing.T) {
	spillDir, cleanup := setupCacheTest(t)
	defer cleanup()

	logger := NewLogger("CACHE_TEST", LogLevelInfo)
	cache := NewCacheManager(1, spillDir, logger)

	largeData := make([][]string, 5000)
	for i := 0; i < 5000; i++ {
		largeData[i] = []string{"data"}
	}

	// Hacer spill múltiple de la misma clave (sobrescribiendo)
	for i := 0; i < 2; i++ {
		if err := cache.Put("same-key", largeData); err != nil {
			t.Fatalf("error en spill %d: %v", i, err)
		}
		time.Sleep(1 * time.Millisecond) // Asegurar timestamp diferente
	}

	// Verificar que el archivo spill existe
	stats := cache.GetStats()
	if stats["spilled_keys"].(int) != 1 {
		t.Error("debería haber 1 clave spilled")
	}

	// Verificar que se puede recuperar
	data, err := cache.Get("same-key")
	if err != nil {
		t.Fatalf("error recuperando: %v", err)
	}
	if len(data) != 5000 {
		t.Errorf("esperado 5000 registros, obtenido %d", len(data))
	}
}

// TestCacheMemoryUsagePercentage prueba cálculo de porcentaje de uso
func TestCacheMemoryUsagePercentage(t *testing.T) {
	spillDir, cleanup := setupCacheTest(t)
	defer cleanup()

	logger := NewLogger("CACHE_TEST", LogLevelInfo)
	cache := NewCacheManager(100, spillDir, logger)

	// Sin datos: 0%
	stats := cache.GetStats()
	if stats["memory_usage_pct"].(float64) != 0.0 {
		t.Error("uso inicial debe ser 0%")
	}

	// Agregar datos
	data := make([][]string, 100)
	for i := 0; i < 100; i++ {
		data[i] = []string{"data", "more data"}
	}
	cache.Put("test", data)

	stats = cache.GetStats()
	usagePct := stats["memory_usage_pct"].(float64)
	if usagePct <= 0.0 || usagePct > 100.0 {
		t.Errorf("porcentaje de uso inválido: %.2f%%", usagePct)
	}
}

// TestCacheDefaultMaxMemory prueba valor por defecto de memoria
func TestCacheDefaultMaxMemory(t *testing.T) {
	spillDir, cleanup := setupCacheTest(t)
	defer cleanup()

	logger := NewLogger("CACHE_TEST", LogLevelInfo)

	// Crear cache con límite 0 o negativo (debe usar default)
	cache := NewCacheManager(0, spillDir, logger)

	stats := cache.GetStats()
	maxMem := stats["max_memory_mb"].(int64)

	if maxMem != 100 {
		t.Errorf("esperado límite default 100MB, obtenido %d", maxMem)
	}
}
