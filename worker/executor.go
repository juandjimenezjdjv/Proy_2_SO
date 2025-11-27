package main

import (
	"encoding/csv"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/juandjimenezjdjv/Proy_2_SO/common"
)

// ExecuteTask ejecuta una tarea (función de paquete para uso desde main)
func ExecuteTask(task *common.Task) error {
	executor := NewExecutor("/app/data", "/app/results", "/app/temp")
	return executor.ExecuteTask(task)
}

// Executor ejecuta operadores sobre datos
type Executor struct {
	dataDir    string
	resultsDir string
	tempDir    string
	cache      *common.CacheManager
}

// NewExecutor crea un nuevo ejecutor
func NewExecutor(dataDir, resultsDir, tempDir string) *Executor {
	// Obtener límite de memoria desde variable de entorno o usar default
	maxMemoryMB := int64(100) // Default: 100MB
	if envMem := os.Getenv("MAX_MEMORY_MB"); envMem != "" {
		if parsedMem, err := fmt.Sscanf(envMem, "%d", &maxMemoryMB); err == nil && parsedMem == 1 {
			// Usar valor parseado
		}
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "INFO"
	}
	logger := common.NewLogger("EXECUTOR", common.LogLevel(logLevel))
	spillDir := filepath.Join(tempDir, "spill")

	return &Executor{
		dataDir:    dataDir,
		resultsDir: resultsDir,
		tempDir:    tempDir,
		cache:      common.NewCacheManager(maxMemoryMB, spillDir, logger),
	}
}

// ExecuteTask ejecuta una tarea según su operador
func (e *Executor) ExecuteTask(task *common.Task) error {
	switch task.Operator {
	case common.OpReadCSV:
		return e.executeReadCSV(task)
	case common.OpMap:
		return e.executeMap(task)
	case common.OpFilter:
		return e.executeFilter(task)
	case common.OpFlatMap:
		return e.executeFlatMap(task)
	case common.OpReduceByKey:
		return e.executeReduceByKey(task)
	case common.OpAggregate:
		return e.executeAggregate(task)
	case common.OpJoin:
		return e.executeJoin(task)
	default:
		return fmt.Errorf("operador desconocido: %s", task.Operator)
	}
}

// executeReadCSV lee un archivo CSV y lo guarda como particiones
func (e *Executor) executeReadCSV(task *common.Task) error {
	if len(task.InputPaths) == 0 {
		return fmt.Errorf("OpReadCSV requiere al menos un archivo de entrada")
	}

	inputPath := filepath.Join(e.dataDir, task.InputPaths[0])
	outputPath := e.getOutputPath(task)

	// Leer archivo CSV
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("error abriendo archivo: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("error leyendo CSV: %w", err)
	}

	// Filtrar registros según partición (hash-based)
	partitionedRecords := e.partitionRecords(records, task.Partition, task.TotalPartitions)

	// Escribir partición
	return e.writeCSV(outputPath, partitionedRecords)
}

// executeMap aplica una transformación 1-a-1 sobre cada registro
// Ejemplo: convertir columnas a mayúsculas, agregar columnas, etc.
func (e *Executor) executeMap(task *common.Task) error {
	// Leer datos de entrada
	records, err := e.readInputRecords(task)
	if err != nil {
		return err
	}

	// Obtener función de transformación
	fnName := ""
	if fn, ok := task.Params["function"].(string); ok {
		fnName = fn
	}

	// Aplicar transformación según función
	mappedRecords := make([][]string, 0, len(records))

	switch fnName {
	case "lowercase":
		// Convertir a minúsculas
		inputCol := 0 // Default
		if ic, ok := task.Params["input_column"].(string); ok && ic == "word" {
			inputCol = 0 // Primera columna
		}

		for _, record := range records {
			if len(record) > inputCol {
				newRecord := make([]string, len(record))
				copy(newRecord, record)
				newRecord[inputCol] = strings.ToLower(record[inputCol])
				mappedRecords = append(mappedRecords, newRecord)
			}
		}

	case "uppercase":
		// Convertir a mayúsculas
		for _, record := range records {
			if len(record) > 0 {
				newRecord := make([]string, len(record))
				copy(newRecord, record)
				newRecord[0] = strings.ToUpper(record[0])
				mappedRecords = append(mappedRecords, newRecord)
			}
		}

	default:
		// Transformación de ejemplo: convertir segunda columna a mayúsculas
		for _, record := range records {
			if len(record) > 1 {
				newRecord := make([]string, len(record))
				copy(newRecord, record)
				newRecord[1] = strings.ToUpper(record[1])
				mappedRecords = append(mappedRecords, newRecord)
			} else {
				mappedRecords = append(mappedRecords, record)
			}
		}
	}

	// Escribir resultado
	outputPath := e.getOutputPath(task)
	return e.writeCSV(outputPath, mappedRecords)
}

// executeFilter filtra registros según una condición
// Ejemplo: mantener solo registros donde columna[0] > 100
func (e *Executor) executeFilter(task *common.Task) error {
	// Leer datos de entrada
	records, err := e.readInputRecords(task)
	if err != nil {
		return err
	}

	// Aplicar filtro (ejemplo: mantener registros con más de 2 columnas)
	filteredRecords := make([][]string, 0)
	for _, record := range records {
		if len(record) >= 2 && record[1] != "" {
			filteredRecords = append(filteredRecords, record)
		}
	}

	// Escribir resultado
	outputPath := e.getOutputPath(task)
	return e.writeCSV(outputPath, filteredRecords)
}

// executeFlatMap transforma cada registro en 0 o más registros (expansión 1-a-N)
// Ejemplo: tokenizar texto, explotar arrays, generar múltiples outputs por input
func (e *Executor) executeFlatMap(task *common.Task) error {
	// Leer datos de entrada
	records, err := e.readInputRecords(task)
	if err != nil {
		return err
	}

	// Obtener función de transformación desde params
	fnName := "split_words" // Default: tokenización
	if fn, ok := task.Params["function"].(string); ok {
		fnName = fn
	} else if fn, ok := task.Params["fn"].(string); ok {
		fnName = fn
	}

	// Aplicar flat_map según función
	flatMappedRecords := make([][]string, 0)

	switch fnName {
	case "split_words", "tokenize":
		// Tokenizar texto: cada palabra se convierte en un registro separado
		inputColumn := 0 // Default: primera columna
		if ic, ok := task.Params["input_column"].(string); ok {
			// Si input_column es "text", usar primera columna (índice 0)
			if ic == "text" {
				inputColumn = 0
			}
		}

		for _, record := range records {
			if len(record) <= inputColumn {
				continue
			}
			// Obtener el texto de la columna especificada
			text := record[inputColumn]
			words := strings.Fields(text) // Split por espacios

			// Generar un registro por cada palabra
			for _, word := range words {
				if word != "" {
					// Limpiar puntuación básica
					word = strings.Trim(word, ".,;:!?()[]{}\"'")
					if word != "" {
						flatMappedRecords = append(flatMappedRecords, []string{word})
					}
				}
			}
		}

	case "split_delimiter":
		// Split por delimitador personalizado
		delimiter := ","
		if d, ok := task.Params["delimiter"].(string); ok {
			delimiter = d
		}
		for _, record := range records {
			if len(record) < 2 {
				continue
			}
			text := record[1]
			parts := strings.Split(text, delimiter)
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" {
					flatMappedRecords = append(flatMappedRecords, []string{part})
				}
			}
		}

	case "explode_array":
		// Explotar array: cada elemento se convierte en registro
		for _, record := range records {
			for _, value := range record {
				if value != "" {
					flatMappedRecords = append(flatMappedRecords, []string{value})
				}
			}
		}

	default:
		// Por defecto: cada registro genera un registro (comportamiento de map)
		for _, record := range records {
			if len(record) > 0 {
				flatMappedRecords = append(flatMappedRecords, record)
			}
		}
	}

	// Escribir resultado
	outputPath := e.getOutputPath(task)
	return e.writeCSV(outputPath, flatMappedRecords)
}

// executeAggregate agrupa registros por clave y aplica función de agregación
// Soporta: count, sum, avg, min, max
// Params.Function especifica la función (default: count)
// Params.ValueColumn especifica la columna a agregar (default: 1)
func (e *Executor) executeAggregate(task *common.Task) error {
	// Leer datos de entrada
	records, err := e.readInputRecords(task)
	if err != nil {
		return err
	}

	// Obtener función de agregación (default: count)
	aggFunc := "count"
	if task.Params != nil {
		if fn, ok := task.Params["function"].(string); ok {
			aggFunc = fn
		}
	}

	// Obtener columna de valor (default: 1)
	valueCol := 1
	if task.Params != nil {
		if col, ok := task.Params["value_column"].(float64); ok {
			valueCol = int(col)
		}
	}

	// Agrupar por primera columna (clave)
	groups := make(map[string][][]string)
	for _, record := range records {
		if len(record) > 0 {
			key := record[0]
			groups[key] = append(groups[key], record)
		}
	}

	// Aplicar función de agregación
	aggregatedRecords := make([][]string, 0, len(groups))
	for key, values := range groups {
		var result string
		switch aggFunc {
		case "count":
			result = fmt.Sprintf("%d", len(values))
		case "sum":
			sum := 0.0
			for _, record := range values {
				if len(record) > valueCol {
					if val, err := strconv.ParseFloat(record[valueCol], 64); err == nil {
						sum += val
					}
				}
			}
			result = fmt.Sprintf("%.2f", sum)
		case "avg":
			sum := 0.0
			count := 0
			for _, record := range values {
				if len(record) > valueCol {
					if val, err := strconv.ParseFloat(record[valueCol], 64); err == nil {
						sum += val
						count++
					}
				}
			}
			if count > 0 {
				result = fmt.Sprintf("%.2f", sum/float64(count))
			} else {
				result = "0.00"
			}
		case "min":
			min := math.MaxFloat64
			for _, record := range values {
				if len(record) > valueCol {
					if val, err := strconv.ParseFloat(record[valueCol], 64); err == nil {
						if val < min {
							min = val
						}
					}
				}
			}
			if min != math.MaxFloat64 {
				result = fmt.Sprintf("%.2f", min)
			} else {
				result = "0.00"
			}
		case "max":
			max := -math.MaxFloat64
			for _, record := range values {
				if len(record) > valueCol {
					if val, err := strconv.ParseFloat(record[valueCol], 64); err == nil {
						if val > max {
							max = val
						}
					}
				}
			}
			if max != -math.MaxFloat64 {
				result = fmt.Sprintf("%.2f", max)
			} else {
				result = "0.00"
			}
		default:
			// Default a count
			result = fmt.Sprintf("%d", len(values))
		}
		aggregatedRecords = append(aggregatedRecords, []string{key, result})
	}

	// Escribir resultado
	outputPath := e.getOutputPath(task)
	return e.writeCSV(outputPath, aggregatedRecords)
}

// executeReduceByKey agrupa registros por clave y aplica agregación
// Ejemplo: contar ocurrencias, sumar valores, etc.
func (e *Executor) executeReduceByKey(task *common.Task) error {
	// Leer datos de entrada
	records, err := e.readInputRecords(task)
	if err != nil {
		return err
	}

	// Agrupar por primera columna (clave)
	groups := make(map[string][][]string)
	for _, record := range records {
		if len(record) > 0 {
			key := record[0]
			groups[key] = append(groups[key], record)
		}
	}

	// Reducir: contar ocurrencias por clave
	reducedRecords := make([][]string, 0, len(groups))
	for key, values := range groups {
		count := fmt.Sprintf("%d", len(values))
		reducedRecords = append(reducedRecords, []string{key, count})
	}

	// Escribir resultado
	outputPath := e.getOutputPath(task)
	return e.writeCSV(outputPath, reducedRecords)
}

// executeJoin une dos datasets por una clave común
func (e *Executor) executeJoin(task *common.Task) error {
	if len(task.InputPaths) < 2 {
		return fmt.Errorf("OpJoin requiere al menos 2 archivos de entrada")
	}

	// Leer primer dataset
	records1, err := e.readCSVFile(task.InputPaths[0])
	if err != nil {
		return fmt.Errorf("error leyendo primer dataset: %w", err)
	}

	// Leer segundo dataset
	records2, err := e.readCSVFile(task.InputPaths[1])
	if err != nil {
		return fmt.Errorf("error leyendo segundo dataset: %w", err)
	}

	// Indexar segundo dataset por clave (primera columna)
	index := make(map[string][][]string)
	for _, record := range records2 {
		if len(record) > 0 {
			key := record[0]
			index[key] = append(index[key], record)
		}
	}

	// Hacer join
	joinedRecords := make([][]string, 0)
	for _, record1 := range records1 {
		if len(record1) > 0 {
			key := record1[0]
			if matches, exists := index[key]; exists {
				for _, record2 := range matches {
					// Combinar registros (eliminar clave duplicada del segundo)
					joined := append([]string{}, record1...)
					if len(record2) > 1 {
						joined = append(joined, record2[1:]...)
					}
					joinedRecords = append(joinedRecords, joined)
				}
			}
		}
	}

	// Escribir resultado
	outputPath := e.getOutputPath(task)
	return e.writeCSV(outputPath, joinedRecords)
}

// partitionRecords divide registros usando hash de la primera columna
func (e *Executor) partitionRecords(records [][]string, partition, totalPartitions int) [][]string {
	if totalPartitions <= 1 {
		return records
	}

	filtered := make([][]string, 0)
	for _, record := range records {
		if len(record) > 0 {
			hash := hashString(record[0])
			if int(hash)%totalPartitions == partition {
				filtered = append(filtered, record)
			}
		}
	}
	return filtered
}

// readInputRecords lee registros de los archivos de entrada de una tarea
func (e *Executor) readInputRecords(task *common.Task) ([][]string, error) {
	allRecords := make([][]string, 0)

	// Si hay InputPaths explícitos, usarlos
	if len(task.InputPaths) > 0 {
		for _, inputPath := range task.InputPaths {
			records, err := e.readCSVFile(inputPath)
			if err != nil {
				return nil, fmt.Errorf("error leyendo %s: %w", inputPath, err)
			}
			allRecords = append(allRecords, records...)
		}
		return allRecords, nil
	}

	// Si no hay InputPaths pero hay Dependencies, buscar archivos temp de las tareas padre
	if len(task.Dependencies) > 0 {
		// Buscar todos los archivos temp que empiecen con "{jobID}-task-*-part-"
		pattern := fmt.Sprintf("%s-task-*-part-*.csv", task.JobID)
		matches, err := filepath.Glob(filepath.Join(e.tempDir, pattern))
		if err != nil {
			return nil, fmt.Errorf("error buscando archivos de dependencias: %w", err)
		}

		// Leer todos los archivos que corresponden a las tareas anteriores
		for _, matchPath := range matches {
			file, err := os.Open(matchPath)
			if err != nil {
				continue
			}
			reader := csv.NewReader(file)
			records, err := reader.ReadAll()
			file.Close()
			if err != nil {
				continue
			}
			allRecords = append(allRecords, records...)
		}

		if len(allRecords) == 0 {
			return nil, fmt.Errorf("no se encontraron datos de las dependencias: %v", task.Dependencies)
		}
		return allRecords, nil
	}

	return nil, fmt.Errorf("no hay InputPaths ni Dependencies configurados")
}

// readCSVFile lee un archivo CSV completo
func (e *Executor) readCSVFile(relativePath string) ([][]string, error) {
	// Normalizar path (quitar prefijo temp/ o results/ si existe)
	normalizedPath := relativePath
	var baseDir string

	if strings.HasPrefix(relativePath, "temp/") {
		normalizedPath = strings.TrimPrefix(relativePath, "temp/")
		baseDir = e.tempDir
	} else if strings.HasPrefix(relativePath, "results/") {
		normalizedPath = strings.TrimPrefix(relativePath, "results/")
		baseDir = e.resultsDir
	} else {
		baseDir = e.dataDir
	}

	// Buscar archivo completo primero
	fullPath := filepath.Join(baseDir, normalizedPath)
	if file, err := os.Open(fullPath); err == nil {
		defer file.Close()
		reader := csv.NewReader(file)
		return reader.ReadAll()
	}

	// Si no existe, buscar particiones (file-part-0.csv, file-part-1.csv, etc.)
	ext := filepath.Ext(normalizedPath)
	base := strings.TrimSuffix(normalizedPath, ext)

	allRecords := make([][]string, 0)
	partitionIdx := 0
	foundAny := false

	for {
		partPath := filepath.Join(baseDir, fmt.Sprintf("%s-part-%d%s", base, partitionIdx, ext))
		file, err := os.Open(partPath)
		if err != nil {
			break // No más particiones
		}
		foundAny = true

		reader := csv.NewReader(file)
		records, err := reader.ReadAll()
		file.Close()

		if err != nil {
			return nil, fmt.Errorf("error leyendo partición %d: %w", partitionIdx, err)
		}

		// Skip header en particiones subsecuentes
		if partitionIdx > 0 && len(records) > 0 {
			records = records[1:]
		}

		allRecords = append(allRecords, records...)
		partitionIdx++
	}

	if !foundAny {
		return nil, fmt.Errorf("archivo no encontrado: %s", relativePath)
	}

	return allRecords, nil
}

// writeCSV escribe registros a un archivo CSV
func (e *Executor) writeCSV(path string, records [][]string) error {
	// Crear directorio si no existe
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creando directorio: %w", err)
	}

	// Crear archivo
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creando archivo: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Escribir registros
	for _, record := range records {
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("error escribiendo registro: %w", err)
		}
	}

	return nil
}

// getOutputPath construye la ruta de salida para una tarea
func (e *Executor) getOutputPath(task *common.Task) string {
	// Si hay path de salida explícito, usarlo
	if task.OutputPath != "" {
		var basePath string
		outputPath := task.OutputPath

		// Determinar directorio base según ruta
		if strings.HasPrefix(outputPath, "temp/") {
			outputPath = strings.TrimPrefix(outputPath, "temp/")
			basePath = e.tempDir
		} else if strings.HasPrefix(outputPath, "results/") {
			outputPath = strings.TrimPrefix(outputPath, "results/")
			basePath = e.resultsDir
		} else {
			// Por defecto usar temp para rutas sin prefijo explícito
			basePath = e.tempDir
		}

		if task.TotalPartitions > 1 {
			// Agregar sufijo de partición
			ext := filepath.Ext(outputPath)
			base := strings.TrimSuffix(outputPath, ext)
			return filepath.Join(basePath, fmt.Sprintf("%s-part-%d%s", base, task.Partition, ext))
		}
		return filepath.Join(basePath, outputPath)
	}

	// Path por defecto basado en task ID
	filename := fmt.Sprintf("%s.csv", task.ID)
	return filepath.Join(e.tempDir, filename)
}

// hashString calcula hash de un string para particionamiento
func hashString(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}
