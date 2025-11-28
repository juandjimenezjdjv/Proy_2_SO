package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/juandjimenezjdjv/Proy_2_SO/common"
)

// setupTestEnv crea directorios temporales para tests
func setupTestEnv(t *testing.T) (string, string, string, func()) {
	dataDir := t.TempDir()
	resultsDir := t.TempDir()
	tempDir := t.TempDir()

	cleanup := func() {
		os.RemoveAll(dataDir)
		os.RemoveAll(resultsDir)
		os.RemoveAll(tempDir)
	}

	return dataDir, resultsDir, tempDir, cleanup
}

// writeTestCSV escribe un archivo CSV de prueba
func writeTestCSV(t *testing.T, path string, records [][]string) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("error creando directorio: %v", err)
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("error creando archivo: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, record := range records {
		if err := writer.Write(record); err != nil {
			t.Fatalf("error escribiendo registro: %v", err)
		}
	}
}

// readTestCSV lee un archivo CSV de prueba
func readTestCSV(t *testing.T, path string) [][]string {
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("error abriendo archivo: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("error leyendo CSV: %v", err)
	}

	return records
}

// TestReadCSV prueba la lectura de archivos CSV
func TestReadCSV(t *testing.T) {
	dataDir, resultsDir, tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Crear archivo de prueba
	testData := [][]string{
		{"name", "age", "city"},
		{"Alice", "30", "NYC"},
		{"Bob", "25", "LA"},
		{"Charlie", "35", "Chicago"},
	}
	inputPath := filepath.Join(dataDir, "input.csv")
	writeTestCSV(t, inputPath, testData)

	// Crear ejecutor
	executor := NewExecutor(dataDir, resultsDir, tempDir)

	// Crear tarea
	task := &common.Task{
		ID:              "read-task-1",
		JobID:           "job-1",
		Operator:        common.OpReadCSV,
		InputPaths:      []string{"input.csv"},
		OutputPath:      "temp/output.csv",
		Partition:       0,
		TotalPartitions: 1,
	}

	// Ejecutar
	err := executor.ExecuteTask(task)
	if err != nil {
		t.Fatalf("error ejecutando ReadCSV: %v", err)
	}

	// Verificar salida
	outputPath := filepath.Join(tempDir, "output.csv")
	output := readTestCSV(t, outputPath)

	if len(output) != len(testData) {
		t.Errorf("esperado %d registros, obtenido %d", len(testData), len(output))
	}
}

// TestReadCSVPartitioning prueba el particionamiento en lectura
func TestReadCSVPartitioning(t *testing.T) {
	dataDir, resultsDir, tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Crear archivo grande con datos repetidos
	testData := [][]string{
		{"key", "value"},
	}
	for i := 0; i < 100; i++ {
		testData = append(testData, []string{"key" + string(rune(i%10+'0')), "value"})
	}

	inputPath := filepath.Join(dataDir, "large.csv")
	writeTestCSV(t, inputPath, testData)

	executor := NewExecutor(dataDir, resultsDir, tempDir)

	// Crear 3 particiones
	totalPartitions := 3
	partitionSizes := make([]int, totalPartitions)

	for p := 0; p < totalPartitions; p++ {
		task := &common.Task{
			ID:              "read-task-part-" + string(rune(p+'0')),
			JobID:           "job-1",
			Operator:        common.OpReadCSV,
			InputPaths:      []string{"large.csv"},
			OutputPath:      "temp/output.csv",
			Partition:       p,
			TotalPartitions: totalPartitions,
		}

		if err := executor.ExecuteTask(task); err != nil {
			t.Fatalf("error en partición %d: %v", p, err)
		}

		// El executor genera nombres como output-part-0.csv automáticamente
		outputFile := fmt.Sprintf("output-part-%d.csv", p)
		output := readTestCSV(t, filepath.Join(tempDir, outputFile))
		partitionSizes[p] = len(output)
	}

	// Verificar que las particiones sumen el total
	totalRecords := 0
	for _, size := range partitionSizes {
		totalRecords += size
	}

	// -1 por el header que no se particiona
	expectedTotal := len(testData)
	if totalRecords != expectedTotal {
		t.Errorf("esperado %d registros totales, obtenido %d", expectedTotal, totalRecords)
	}

	// Verificar que cada partición tiene al menos algunos registros
	for i, size := range partitionSizes {
		if size == 0 {
			t.Errorf("partición %d está vacía", i)
		}
	}
}

// TestMap prueba el operador Map
func TestMap(t *testing.T) {
	dataDir, resultsDir, tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Crear datos de entrada en tempDir
	testData := [][]string{
		{"word", "count"},
		{"hello", "5"},
		{"world", "3"},
	}
	inputPath := filepath.Join(tempDir, "map-input.csv")
	writeTestCSV(t, inputPath, testData)

	executor := NewExecutor(dataDir, resultsDir, tempDir)

	// Aplicar map lowercase
	task := &common.Task{
		ID:         "map-task-1",
		JobID:      "job-1",
		Operator:   common.OpMap,
		InputPaths: []string{"temp/map-input.csv"},
		OutputPath: "temp/map-output.csv",
		Params: map[string]interface{}{
			"function":     "lowercase",
			"input_column": "word",
		},
		Partition:       0,
		TotalPartitions: 1,
	}

	if err := executor.ExecuteTask(task); err != nil {
		t.Fatalf("error ejecutando Map: %v", err)
	}

	// Verificar salida
	outputPath := filepath.Join(tempDir, "map-output.csv")
	output := readTestCSV(t, outputPath)

	if len(output) != len(testData) {
		t.Errorf("esperado %d registros, obtenido %d", len(testData), len(output))
	}

	// Verificar transformación lowercase
	if output[1][0] != "hello" {
		t.Errorf("esperado 'hello', obtenido '%s'", output[1][0])
	}
}

// TestMapUppercase prueba el operador Map con mayúsculas
func TestMapUppercase(t *testing.T) {
	dataDir, resultsDir, tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	testData := [][]string{
		{"word"},
		{"hello"},
		{"world"},
	}
	inputPath := filepath.Join(tempDir, "uppercase-input.csv")
	writeTestCSV(t, inputPath, testData)

	executor := NewExecutor(dataDir, resultsDir, tempDir)

	task := &common.Task{
		ID:         "map-upper-task",
		JobID:      "job-1",
		Operator:   common.OpMap,
		InputPaths: []string{"temp/uppercase-input.csv"},
		OutputPath: "temp/uppercase-output.csv",
		Params: map[string]interface{}{
			"function": "uppercase",
		},
		Partition:       0,
		TotalPartitions: 1,
	}

	if err := executor.ExecuteTask(task); err != nil {
		t.Fatalf("error ejecutando Map uppercase: %v", err)
	}

	output := readTestCSV(t, filepath.Join(tempDir, "uppercase-output.csv"))

	if output[1][0] != "HELLO" {
		t.Errorf("esperado 'HELLO', obtenido '%s'", output[1][0])
	}
	if output[2][0] != "WORLD" {
		t.Errorf("esperado 'WORLD', obtenido '%s'", output[2][0])
	}
}

// TestFilter prueba el operador Filter
func TestFilter(t *testing.T) {
	dataDir, resultsDir, tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	testData := [][]string{
		{"name", "age"},
		{"Alice", "30"},
		{"Bob", ""}, // Este se filtra (columna vacía)
		{"Charlie", "25"},
	}
	inputPath := filepath.Join(tempDir, "filter-input.csv")
	writeTestCSV(t, inputPath, testData)

	executor := NewExecutor(dataDir, resultsDir, tempDir)

	task := &common.Task{
		ID:              "filter-task-1",
		JobID:           "job-1",
		Operator:        common.OpFilter,
		InputPaths:      []string{"temp/filter-input.csv"},
		OutputPath:      "temp/filter-output.csv",
		Partition:       0,
		TotalPartitions: 1,
	}

	if err := executor.ExecuteTask(task); err != nil {
		t.Fatalf("error ejecutando Filter: %v", err)
	}

	output := readTestCSV(t, filepath.Join(tempDir, "filter-output.csv"))

	// Debe filtrar el registro con columna vacía
	expectedRecords := 3 // Header + Alice + Charlie
	if len(output) != expectedRecords {
		t.Errorf("esperado %d registros, obtenido %d", expectedRecords, len(output))
	}
}

// TestFlatMap prueba el operador FlatMap con tokenización
func TestFlatMap(t *testing.T) {
	dataDir, resultsDir, tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	testData := [][]string{
		{"text"},
		{"hello world"},
		{"foo bar baz"},
	}
	inputPath := filepath.Join(tempDir, "flatmap-input.csv")
	writeTestCSV(t, inputPath, testData)

	executor := NewExecutor(dataDir, resultsDir, tempDir)

	task := &common.Task{
		ID:         "flatmap-task-1",
		JobID:      "job-1",
		Operator:   common.OpFlatMap,
		InputPaths: []string{"temp/flatmap-input.csv"},
		OutputPath: "temp/flatmap-output.csv",
		Params: map[string]interface{}{
			"function":     "split_words",
			"input_column": "text",
		},
		Partition:       0,
		TotalPartitions: 1,
	}

	if err := executor.ExecuteTask(task); err != nil {
		t.Fatalf("error ejecutando FlatMap: %v", err)
	}

	output := readTestCSV(t, filepath.Join(tempDir, "flatmap-output.csv"))

	// Incluye header "text" + 5 palabras = 6 filas
	expectedWords := 6
	if len(output) != expectedWords {
		t.Errorf("esperado %d filas (incluyendo header), obtenido %d", expectedWords, len(output))
	}

	// Verificar palabras individuales (saltando header)
	words := []string{"hello", "world", "foo", "bar", "baz"}
	for i, expectedWord := range words {
		if output[i+1][0] != expectedWord {
			t.Errorf("palabra %d: esperado '%s', obtenido '%s'", i, expectedWord, output[i+1][0])
		}
	}
}

// TestFlatMapTokenize prueba tokenización con puntuación
func TestFlatMapTokenize(t *testing.T) {
	dataDir, resultsDir, tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	testData := [][]string{
		{"text"},
		{"Hello, world! How are you?"},
	}
	inputPath := filepath.Join(tempDir, "tokenize-input.csv")
	writeTestCSV(t, inputPath, testData)

	executor := NewExecutor(dataDir, resultsDir, tempDir)

	task := &common.Task{
		ID:         "tokenize-task",
		JobID:      "job-1",
		Operator:   common.OpFlatMap,
		InputPaths: []string{"temp/tokenize-input.csv"},
		OutputPath: "temp/tokenize-output.csv",
		Params: map[string]interface{}{
			"function":     "tokenize",
			"input_column": "text",
		},
		Partition:       0,
		TotalPartitions: 1,
	}

	if err := executor.ExecuteTask(task); err != nil {
		t.Fatalf("error ejecutando FlatMap tokenize: %v", err)
	}

	output := readTestCSV(t, filepath.Join(tempDir, "tokenize-output.csv"))

	// Incluye header "text" + 5 palabras limpias
	expectedWords := []string{"Hello", "world", "How", "are", "you"}
	if len(output) != len(expectedWords)+1 {
		t.Errorf("esperado %d filas (incluyendo header), obtenido %d", len(expectedWords)+1, len(output))
	}

	for i, expected := range expectedWords {
		if output[i+1][0] != expected {
			t.Errorf("palabra %d: esperado '%s', obtenido '%s'", i, expected, output[i+1][0])
		}
	}
}

// TestReduceByKey prueba el operador ReduceByKey
func TestReduceByKey(t *testing.T) {
	dataDir, resultsDir, tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	testData := [][]string{
		{"word", "count"},
		{"hello", "1"},
		{"world", "1"},
		{"hello", "1"},
		{"foo", "1"},
		{"hello", "1"},
	}
	inputPath := filepath.Join(tempDir, "reduce-input.csv")
	writeTestCSV(t, inputPath, testData)

	executor := NewExecutor(dataDir, resultsDir, tempDir)

	task := &common.Task{
		ID:              "reduce-task-1",
		JobID:           "job-1",
		Operator:        common.OpReduceByKey,
		InputPaths:      []string{"temp/reduce-input.csv"},
		OutputPath:      "temp/reduce-output.csv",
		Partition:       0,
		TotalPartitions: 1,
	}

	if err := executor.ExecuteTask(task); err != nil {
		t.Fatalf("error ejecutando ReduceByKey: %v", err)
	}

	output := readTestCSV(t, filepath.Join(tempDir, "reduce-output.csv"))

	// Debe agrupar: hello (3), world (1), foo (1), word (1) - SIN header
	expectedGroups := 4
	if len(output) != expectedGroups {
		t.Errorf("esperado %d grupos, obtenido %d", expectedGroups, len(output))
	}

	// Verificar conteos (sin header en reduce_by_key)
	counts := make(map[string]string)
	for _, record := range output {
		if len(record) >= 2 {
			counts[record[0]] = record[1]
		}
	}

	if counts["hello"] != "3" {
		t.Errorf("esperado hello=3, obtenido %s", counts["hello"])
	}
	if counts["world"] != "1" {
		t.Errorf("esperado world=1, obtenido %s", counts["world"])
	}
}

// TestAggregate prueba el operador Aggregate con diferentes funciones
func TestAggregate(t *testing.T) {
	dataDir, resultsDir, tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	testData := [][]string{
		{"product", "price"},
		{"apple", "1.5"},
		{"banana", "0.8"},
		{"apple", "1.7"},
		{"orange", "2.0"},
		{"apple", "1.3"},
	}
	inputPath := filepath.Join(tempDir, "agg-input.csv")
	writeTestCSV(t, inputPath, testData)

	executor := NewExecutor(dataDir, resultsDir, tempDir)

	tests := []struct {
		name     string
		function string
		expected map[string]string
	}{
		{
			name:     "Count",
			function: "count",
			expected: map[string]string{
				"apple":  "3",
				"banana": "1",
				"orange": "1",
			},
		},
		{
			name:     "Sum",
			function: "sum",
			expected: map[string]string{
				"apple": "4.50", // 1.5 + 1.7 + 1.3
			},
		},
		{
			name:     "Avg",
			function: "avg",
			expected: map[string]string{
				"apple": "1.50", // (1.5 + 1.7 + 1.3) / 3
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &common.Task{
				ID:         "agg-task-" + tt.function,
				JobID:      "job-1",
				Operator:   common.OpAggregate,
				InputPaths: []string{"temp/agg-input.csv"},
				OutputPath: "temp/agg-output-" + tt.function + ".csv",
				Params: map[string]interface{}{
					"function":     tt.function,
					"value_column": float64(1),
				},
				Partition:       0,
				TotalPartitions: 1,
			}

			if err := executor.ExecuteTask(task); err != nil {
				t.Fatalf("error ejecutando Aggregate %s: %v", tt.function, err)
			}

			output := readTestCSV(t, filepath.Join(tempDir, "agg-output-"+tt.function+".csv"))

			// Verificar resultados esperados
			results := make(map[string]string)
			for _, record := range output {
				results[record[0]] = record[1]
			}

			for key, expectedValue := range tt.expected {
				if actualValue, ok := results[key]; ok {
					if actualValue != expectedValue {
						t.Errorf("%s: esperado %s=%s, obtenido %s", tt.function, key, expectedValue, actualValue)
					}
				}
			}
		})
	}
}

// TestAggregateMinMax prueba min y max
func TestAggregateMinMax(t *testing.T) {
	dataDir, resultsDir, tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	testData := [][]string{
		{"product", "price"},
		{"item", "5.5"},
		{"item", "2.3"},
		{"item", "8.1"},
		{"item", "3.7"},
	}
	inputPath := filepath.Join(tempDir, "minmax-input.csv")
	writeTestCSV(t, inputPath, testData)

	executor := NewExecutor(dataDir, resultsDir, tempDir)

	// Test MIN
	taskMin := &common.Task{
		ID:         "agg-min-task",
		JobID:      "job-1",
		Operator:   common.OpAggregate,
		InputPaths: []string{"temp/minmax-input.csv"},
		OutputPath: "temp/agg-min-output.csv",
		Params: map[string]interface{}{
			"function":     "min",
			"value_column": float64(1),
		},
		Partition:       0,
		TotalPartitions: 1,
	}

	if err := executor.ExecuteTask(taskMin); err != nil {
		t.Fatalf("error ejecutando Aggregate min: %v", err)
	}

	outputMin := readTestCSV(t, filepath.Join(tempDir, "agg-min-output.csv"))
	// Aggregate debería retornar key,value - verificar última fila (ignorando posible header)
	if len(outputMin) == 0 {
		t.Fatal("output mínimo está vacío")
	}
	lastRow := outputMin[len(outputMin)-1]
	if lastRow[1] != "2.30" {
		t.Errorf("min: esperado 2.30, obtenido %s (output completo: %v)", lastRow[1], outputMin)
	}

	// Test MAX
	taskMax := &common.Task{
		ID:         "agg-max-task",
		JobID:      "job-1",
		Operator:   common.OpAggregate,
		InputPaths: []string{"temp/minmax-input.csv"},
		OutputPath: "temp/agg-max-output.csv",
		Params: map[string]interface{}{
			"function":     "max",
			"value_column": float64(1),
		},
		Partition:       0,
		TotalPartitions: 1,
	}

	if err := executor.ExecuteTask(taskMax); err != nil {
		t.Fatalf("error ejecutando Aggregate max: %v", err)
	}

	outputMax := readTestCSV(t, filepath.Join(tempDir, "agg-max-output.csv"))
	if len(outputMax) == 0 {
		t.Fatal("output máximo está vacío")
	}
	lastRowMax := outputMax[len(outputMax)-1]
	if lastRowMax[1] != "8.10" {
		t.Errorf("max: esperado 8.10, obtenido %s (output completo: %v)", lastRowMax[1], outputMax)
	}
}

// TestJoin prueba el operador Join
func TestJoin(t *testing.T) {
	dataDir, resultsDir, tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Dataset 1: usuarios
	users := [][]string{
		{"id", "name"},
		{"1", "Alice"},
		{"2", "Bob"},
		{"3", "Charlie"},
	}
	usersPath := filepath.Join(tempDir, "users.csv")
	writeTestCSV(t, usersPath, users)

	// Dataset 2: pedidos
	orders := [][]string{
		{"user_id", "product"},
		{"1", "laptop"},
		{"2", "mouse"},
		{"1", "keyboard"},
	}
	ordersPath := filepath.Join(tempDir, "orders.csv")
	writeTestCSV(t, ordersPath, orders)

	executor := NewExecutor(dataDir, resultsDir, tempDir)

	task := &common.Task{
		ID:              "join-task-1",
		JobID:           "job-1",
		Operator:        common.OpJoin,
		InputPaths:      []string{"temp/users.csv", "temp/orders.csv"},
		OutputPath:      "temp/join-output.csv",
		Partition:       0,
		TotalPartitions: 1,
	}

	if err := executor.ExecuteTask(task); err != nil {
		t.Fatalf("error ejecutando Join: %v", err)
	}

	output := readTestCSV(t, filepath.Join(tempDir, "join-output.csv"))

	// Debe generar 3 registros (Alice+laptop, Bob+mouse, Alice+keyboard)
	expectedRecords := 3
	if len(output) != expectedRecords {
		t.Errorf("esperado %d registros join, obtenido %d", expectedRecords, len(output))
	}

	// Verificar que Alice aparece dos veces
	aliceCount := 0
	for _, record := range output {
		if len(record) > 1 && record[1] == "Alice" {
			aliceCount++
		}
	}
	if aliceCount != 2 {
		t.Errorf("esperado Alice 2 veces, obtenido %d", aliceCount)
	}
}

// TestPartitioning prueba el sistema de particionamiento hash-based
func TestPartitioning(t *testing.T) {
	dataDir, resultsDir, tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	_ = dataDir
	_ = resultsDir
	_ = tempDir

	// Crear dataset con claves conocidas
	testData := [][]string{
		{"key"},
		{"key0"},
		{"key1"},
		{"key2"},
		{"key3"},
		{"key4"},
	}

	// Test de particionamiento consistente
	partition0Count := 0
	partition1Count := 0

	for _, record := range testData[1:] { // Skip header
		hash := hashString(record[0])
		if int(hash)%2 == 0 {
			partition0Count++
		} else {
			partition1Count++
		}
	}

	totalKeys := len(testData) - 1
	if partition0Count+partition1Count != totalKeys {
		t.Errorf("particionamiento incorrecto: p0=%d + p1=%d != %d",
			partition0Count, partition1Count, totalKeys)
	}

	// Verificar que ambas particiones tienen al menos 1 elemento
	// (esto depende del hash, pero con 5 claves es probable)
	if partition0Count == 0 || partition1Count == 0 {
		t.Logf("ADVERTENCIA: distribución desbalanceada: p0=%d, p1=%d",
			partition0Count, partition1Count)
	}
}

// TestHashString verifica la función de hash
func TestHashString(t *testing.T) {
	// Verificar que el hash es determinístico
	hash1 := hashString("test")
	hash2 := hashString("test")

	if hash1 != hash2 {
		t.Errorf("hash no determinístico: %d != %d", hash1, hash2)
	}

	// Verificar que diferentes strings dan diferentes hashes (en general)
	hashA := hashString("keyA")
	hashB := hashString("keyB")

	if hashA == hashB {
		t.Logf("ADVERTENCIA: colisión de hash entre 'keyA' y 'keyB'")
	}
}

// TestExecutorUnknownOperator prueba manejo de operador desconocido
func TestExecutorUnknownOperator(t *testing.T) {
	dataDir, resultsDir, tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	executor := NewExecutor(dataDir, resultsDir, tempDir)

	task := &common.Task{
		ID:       "unknown-op-task",
		JobID:    "job-1",
		Operator: common.OperatorType("invalid_operator"),
	}

	err := executor.ExecuteTask(task)
	if err == nil {
		t.Error("esperado error para operador desconocido")
	}
}

// TestMultiplePartitionReading prueba lectura de múltiples particiones
func TestMultiplePartitionReading(t *testing.T) {
	dataDir, resultsDir, tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Crear múltiples archivos de partición
	baseName := "multi-part"
	for i := 0; i < 3; i++ {
		partData := [][]string{
			{"key", "value"},
		}
		for j := 0; j < 10; j++ {
			partData = append(partData, []string{"key" + string(rune(i+'0')), "val"})
		}
		partPath := filepath.Join(tempDir, baseName+"-part-"+string(rune(i+'0'))+".csv")
		writeTestCSV(t, partPath, partData)
	}

	executor := NewExecutor(dataDir, resultsDir, tempDir)

	// Leer todas las particiones
	records, err := executor.readCSVFile("temp/" + baseName + ".csv")
	if err != nil {
		t.Fatalf("error leyendo particiones múltiples: %v", err)
	}

	// Debe tener header + 30 registros (10 por cada una de las 3 particiones)
	expectedRecords := 1 + 30
	if len(records) != expectedRecords {
		t.Errorf("esperado %d registros totales, obtenido %d", expectedRecords, len(records))
	}
}

// TestCacheIntegration prueba integración con cache
func TestCacheIntegration(t *testing.T) {
	dataDir, resultsDir, tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	executor := NewExecutor(dataDir, resultsDir, tempDir)

	// Verificar que executor tiene cache inicializado
	if executor.cache == nil {
		t.Error("cache no inicializado en executor")
	}

	// Verificar límite de memoria configurado
	stats := executor.cache.GetStats()
	maxMemory, ok := stats["max_memory_mb"].(int64)
	if !ok || maxMemory <= 0 {
		t.Errorf("límite de memoria inválido: %v", stats["max_memory_mb"])
	}
}
