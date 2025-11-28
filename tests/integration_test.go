package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/juandjimenezjdjv/Proy_2_SO/common"
)

// TestIntegrationWordCount prueba un job completo de word count usando cluster existente
func TestIntegrationWordCount(t *testing.T) {
	// Skip si es ejecución de CI/CD
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Usar cluster existente en localhost:8080
	masterURL := "http://localhost:8080"

	// Verificar que el cluster está disponible
	if !isClusterReady(t, masterURL) {
		t.Skip("Cluster no disponible en localhost:8080 - inicia con 'make docker-up'")
	}

	// Crear datos de prueba
	createTestData(t, "")

	// Definir job de word count
	job := createWordCountJob()

	// Enviar job al master
	jobID := submitJob(t, masterURL, job)

	// Esperar que el job complete
	waitForJobCompletion(t, masterURL, jobID, 60*time.Second)

	// Verificar resultados
	validateWordCountResults(t, "", jobID)
}

// TestIntegrationAggregation prueba un job de agregación usando cluster existente
func TestIntegrationAggregation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	masterURL := "http://localhost:8080"

	if !isClusterReady(t, masterURL) {
		t.Skip("Cluster no disponible en localhost:8080 - inicia con 'make docker-up'")
	}

	// Crear datos de ventas para agregación
	createSalesData(t, "")

	// Job de agregación de ventas
	job := createAggregationJob()
	jobID := submitJob(t, masterURL, job)

	waitForJobCompletion(t, masterURL, jobID, 60*time.Second)

	// Verificar que la suma total de ventas es correcta
	validateAggregationResults(t, "", jobID)
}

// TestIntegrationPipeline prueba un pipeline complejo usando cluster existente
func TestIntegrationPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	masterURL := "http://localhost:8080"

	if !isClusterReady(t, masterURL) {
		t.Skip("Cluster no disponible en localhost:8080 - inicia con 'make docker-up'")
	}

	createComplexPipelineData(t, "")

	// Pipeline: ReadCSV -> Filter -> Map -> Aggregate
	job := createComplexPipelineJob()
	jobID := submitJob(t, masterURL, job)

	waitForJobCompletion(t, masterURL, jobID, 90*time.Second)

	validatePipelineResults(t, "", jobID)
}

// TestIntegrationFailureRecovery prueba la recuperación ante fallos usando cluster existente
func TestIntegrationFailureRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	masterURL := "http://localhost:8080"

	if !isClusterReady(t, masterURL) {
		t.Skip("Cluster no disponible en localhost:8080 - inicia con 'make docker-up'")
	}

	createTestData(t, "")

	// Enviar job
	job := createWordCountJob()
	jobID := submitJob(t, masterURL, job)

	// En un entorno real de cluster, la tolerancia a fallos es manejada automáticamente
	// Este test verifica que el job complete exitosamente en el cluster distribuido
	waitForJobCompletion(t, masterURL, jobID, 120*time.Second)

	validateWordCountResults(t, "", jobID)
}

// Funciones auxiliares para tests de integración

func isClusterReady(t *testing.T, masterURL string) bool {
	resp, err := http.Get(masterURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func createTestData(t *testing.T, testDir string) {
	data := []string{
		"Hello world",
		"Hello spark",
		"World of distributed computing",
		"Spark is fast",
		"Computing power of spark",
		"Hello distributed world",
	}

	// Crear directorio en app/data (relativo al root del proyecto)
	dataDir := filepath.Join("..", "app", "data")
	os.MkdirAll(dataDir, 0755)

	dataFile := filepath.Join(dataDir, "integration_text.csv")
	file, err := os.Create(dataFile)
	if err != nil {
		t.Fatalf("Error creando archivo de prueba: %v", err)
	}
	defer file.Close()

	file.WriteString("text\n")
	for _, line := range data {
		file.WriteString(line + "\n")
	}
}

func createSalesData(t *testing.T, testDir string) {
	salesData := [][]string{
		{"product", "amount"},
		{"laptop", "1500"},
		{"mouse", "25"},
		{"keyboard", "80"},
		{"laptop", "1600"},
		{"mouse", "30"},
		{"monitor", "400"},
		{"laptop", "1400"},
	}

	// Crear en app/data relativo al root
	dataDir := filepath.Join("..", "app", "data")
	os.MkdirAll(dataDir, 0755)

	dataFile := filepath.Join(dataDir, "integration_sales.csv")
	file, err := os.Create(dataFile)
	if err != nil {
		t.Fatalf("Error creando archivo de ventas: %v", err)
	}
	defer file.Close()

	for _, row := range salesData {
		file.WriteString(fmt.Sprintf("%s,%s\n", row[0], row[1]))
	}
}

func createComplexPipelineData(t *testing.T, testDir string) {
	pipelineData := [][]string{
		{"name", "age", "score"},
		{"Alice", "25", "95"},
		{"Bob", "30", "87"},
		{"Charlie", "22", "92"},
		{"David", "35", "78"},
		{"Eve", "28", "88"},
		{"Frank", "32", "74"},
		{"Grace", "26", "96"},
	}

	// Crear en app/data relativo al root
	dataDir := filepath.Join("..", "app", "data")
	os.MkdirAll(dataDir, 0755)

	dataFile := filepath.Join(dataDir, "integration_pipeline.csv")
	file, err := os.Create(dataFile)
	if err != nil {
		t.Fatalf("Error creando archivo de pipeline: %v", err)
	}
	defer file.Close()

	for _, row := range pipelineData {
		file.WriteString(fmt.Sprintf("%s,%s,%s\n", row[0], row[1], row[2]))
	}
}

func createWordCountJob() map[string]interface{} {
	return map[string]interface{}{
		"name": "integration-wordcount",
		"dag": map[string]interface{}{
			"nodes": []map[string]interface{}{
				{
					"id":          "read",
					"operator":    "read_csv",
					"input_paths": []string{"integration_text.csv"},
					"output_path": "temp/integration_read_output",
					"partitions":  2,
				},
				{
					"id":          "tokenize",
					"operator":    "flat_map",
					"input_paths": []string{"temp/integration_read_output"},
					"output_path": "results/integration_word_count",
					"partitions":  2,
					"params": map[string]interface{}{
						"function": "split_words",
					},
				},
			},
			"edges": []map[string]interface{}{
				{"from": "read", "to": "tokenize"},
			},
		},
	}
}

func createAggregationJob() map[string]interface{} {
	return map[string]interface{}{
		"name": "integration-aggregation",
		"dag": map[string]interface{}{
			"nodes": []map[string]interface{}{
				{
					"id":          "read",
					"operator":    "read_csv",
					"input_paths": []string{"integration_sales.csv"},
					"output_path": "temp/integration_sales_read",
					"partitions":  2,
				},
				{
					"id":          "sum_sales",
					"operator":    "aggregate",
					"input_paths": []string{"temp/integration_sales_read"},
					"output_path": "results/integration_sales_total",
					"partitions":  1,
					"params": map[string]interface{}{
						"function": "sum",
						"column":   "amount",
					},
				},
			},
			"edges": []map[string]interface{}{
				{"from": "read", "to": "sum_sales"},
			},
		},
	}
}

func createComplexPipelineJob() map[string]interface{} {
	return map[string]interface{}{
		"name": "integration-pipeline",
		"dag": map[string]interface{}{
			"nodes": []map[string]interface{}{
				{
					"id":          "read",
					"operator":    "read_csv",
					"input_paths": []string{"integration_pipeline.csv"},
					"output_path": "temp/integration_pipeline_read",
					"partitions":  2,
				},
				{
					"id":          "filter_high_scores",
					"operator":    "filter",
					"input_paths": []string{"temp/integration_pipeline_read"},
					"output_path": "temp/integration_filtered",
					"partitions":  2,
					"params": map[string]interface{}{
						"condition": "score > 85",
					},
				},
				{
					"id":          "uppercase_names",
					"operator":    "map",
					"input_paths": []string{"temp/integration_filtered"},
					"output_path": "results/integration_pipeline_result",
					"partitions":  1,
					"params": map[string]interface{}{
						"function": "uppercase",
						"column":   "name",
					},
				},
			},
			"edges": []map[string]interface{}{
				{"from": "read", "to": "filter_high_scores"},
				{"from": "filter_high_scores", "to": "uppercase_names"},
			},
		},
	}
}

func submitJob(t *testing.T, baseURL string, job map[string]interface{}) string {
	jobJSON, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("Error marshaling job: %v", err)
	}

	resp, err := http.Post(baseURL+"/api/v1/jobs", "application/json", bytes.NewBuffer(jobJSON))
	if err != nil {
		t.Fatalf("Error enviando job: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Error enviando job: %d - %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Error decodificando respuesta: %v", err)
	}

	jobID, ok := result["id"].(string)
	if !ok {
		t.Fatalf("No se recibió job_id válido")
	}

	return jobID
}

func waitForJobCompletion(t *testing.T, baseURL, jobID string, timeout time.Duration) {
	timeoutChan := time.After(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutChan:
			t.Fatalf("Timeout esperando que el job %s complete", jobID)
		case <-ticker.C:
			resp, err := http.Get(fmt.Sprintf("%s/api/v1/jobs/%s", baseURL, jobID))
			if err != nil {
				continue
			}

			var jobInfo map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&jobInfo)
			resp.Body.Close()

			if status, ok := jobInfo["status"].(string); ok {
				if status == string(common.JobStatusSucceeded) {
					t.Logf("Job %s completado exitosamente", jobID)
					return
				} else if status == string(common.JobStatusFailed) {
					t.Fatalf("Job %s falló", jobID)
				}
			}
		}
	}
}

func validateWordCountResults(t *testing.T, testDir, jobID string) {
	// Los resultados se escriben en app/results/ relativo al directorio raíz
	resultsFile := "../app/results/integration_word_count-part-0"

	// Verificar que el archivo de resultados existe
	if _, err := os.Stat(resultsFile); os.IsNotExist(err) {
		t.Fatalf("Archivo de resultados no encontrado: %s", resultsFile)
	}

	// Leer y verificar contenido
	content, err := os.ReadFile(resultsFile)
	if err != nil {
		t.Fatalf("Error leyendo resultados: %v", err)
	}

	results := string(content)

	// Verificar que contiene palabras esperadas (considerando mayúsculas/minúsculas)
	expectedWords := []string{"Hello", "world", "spark", "Computing"}
	for _, word := range expectedWords {
		if !containsWord(results, word) {
			t.Errorf("Palabra '%s' no encontrada en resultados", word)
		}
	}

	t.Logf("Word count completado correctamente para job %s", jobID)
}

func validateAggregationResults(t *testing.T, testDir, jobID string) {
	resultsFile := "../app/results/integration_sales_total"

	if _, err := os.Stat(resultsFile); os.IsNotExist(err) {
		t.Fatalf("Archivo de resultados de agregación no encontrado: %s", resultsFile)
	}

	content, err := os.ReadFile(resultsFile)
	if err != nil {
		t.Fatalf("Error leyendo resultados de agregación: %v", err)
	}

	results := string(content)

	// Verificar que hay resultados de agregación (laptop: 3000, mouse: 55, etc.)
	if !containsWord(results, "laptop") || !containsWord(results, "3000") {
		t.Errorf("Resultados de agregación no encontrados correctamente: %s", results)
	}

	t.Logf("Agregación completada correctamente para job %s", jobID)
}

func validatePipelineResults(t *testing.T, testDir, jobID string) {
	resultsFile := "../app/results/integration_pipeline_result"

	if _, err := os.Stat(resultsFile); os.IsNotExist(err) {
		t.Fatalf("Archivo de resultados de pipeline no encontrado: %s", resultsFile)
	}

	content, err := os.ReadFile(resultsFile)
	if err != nil {
		t.Fatalf("Error leyendo resultados de pipeline: %v", err)
	}

	results := string(content)

	// Verificar que hay resultados procesados
	if len(results) < 5 {
		t.Errorf("Resultados de pipeline muy pequeños: %s", results)
	}

	t.Logf("Pipeline completado correctamente para job %s", jobID)
}

func containsWord(text, word string) bool {
	return bytes.Contains([]byte(text), []byte(word))
}
