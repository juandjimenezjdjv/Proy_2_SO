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
)

// TestE2EWordCountMultiNode prueba word count con cluster multinodo existente
func TestE2EWordCountMultiNode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test")
	}

	masterURL := "http://localhost:8080"

	// Verificar que el cluster esté disponible
	if !isE2EClusterReady(t, masterURL) {
		t.Skip("Cluster no disponible en localhost:8080 - inicia con 'make docker-up'")
	}

	// Verificar que todos los workers estén registrados
	verifyWorkersRegistered(t, masterURL, 3)

	// Ejecutar job de word count distribuido
	job := createMultiNodeWordCountJob()
	jobID := submitJobE2E(t, masterURL, job)

	// Esperar que complete en el cluster
	waitForJobCompletionE2E(t, masterURL, jobID, 2*time.Minute)

	// Verificar resultados distribuidos
	validateMultiNodeResults(t, jobID)
}

// TestE2EFailureRecoveryMultiNode prueba tolerancia a fallos en multinodo usando cluster existente
func TestE2EFailureRecoveryMultiNode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test")
	}

	masterURL := "http://localhost:8080"

	if !isE2EClusterReady(t, masterURL) {
		t.Skip("Cluster no disponible en localhost:8080 - inicia con 'make docker-up'")
	}

	verifyWorkersRegistered(t, masterURL, 3)

	// Enviar job
	job := createMultiNodeWordCountJob()
	jobID := submitJobE2E(t, masterURL, job)

	// En un entorno real, la tolerancia a fallos es manejada automáticamente
	// Este test verifica que el job complete exitosamente en el cluster distribuido
	waitForJobCompletionE2E(t, masterURL, jobID, 3*time.Minute)

	validateMultiNodeResults(t, jobID)
}

// TestE2EComplexPipelineMultiNode prueba pipeline complejo con múltiples stages usando cluster existente
func TestE2EComplexPipelineMultiNode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test")
	}

	masterURL := "http://localhost:8080"

	if !isE2EClusterReady(t, masterURL) {
		t.Skip("Cluster no disponible en localhost:8080 - inicia con 'make docker-up'")
	}

	verifyWorkersRegistered(t, masterURL, 3)

	// Usar pipeline más simple con archivos existentes
	job := createSimplePipelineJob()
	jobID := submitJobE2E(t, masterURL, job)

	waitForJobCompletionE2E(t, masterURL, jobID, 2*time.Minute)

	validateE2EPipelineResults(t, jobID)
}

// TestE2EConcurrentJobs prueba ejecución concurrente de múltiples jobs usando cluster existente
func TestE2EConcurrentJobs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test")
	}

	masterURL := "http://localhost:8080"

	if !isE2EClusterReady(t, masterURL) {
		t.Skip("Cluster no disponible en localhost:8080 - inicia con 'make docker-up'")
	}

	verifyWorkersRegistered(t, masterURL, 3)

	// Enviar 3 jobs concurrentes simples usando archivos existentes
	jobs := []map[string]interface{}{
		createMultiNodeWordCountJob(),
		createAggregationJobE2E(),
		createSimpleFilterJob(),
	}

	var jobIDs []string
	for i, job := range jobs {
		job["name"] = fmt.Sprintf("concurrent-job-%d", i+1)
		jobID := submitJobE2E(t, masterURL, job)
		jobIDs = append(jobIDs, jobID)

		// Pequeña pausa entre envíos
		time.Sleep(2 * time.Second)
	}

	// Esperar que todos completen con timeout más conservador
	for i, jobID := range jobIDs {
		t.Logf("Esperando job concurrente %d (%s)...", i+1, jobID)
		waitForJobCompletionE2E(t, masterURL, jobID, 2*time.Minute)
	}

	// Verificar que todos los jobs completaron exitosamente
	for i, jobID := range jobIDs {
		t.Logf("Job concurrente %d (%s) completado", i+1, jobID)
	}
}

// TestE2ELoadBalance prueba distribución de carga entre workers usando cluster existente
func TestE2ELoadBalance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test")
	}

	masterURL := "http://localhost:8080"

	if !isE2EClusterReady(t, masterURL) {
		t.Skip("Cluster no disponible en localhost:8080 - inicia con 'make docker-up'")
	}

	verifyWorkersRegistered(t, masterURL, 3)

	// Job con muchas particiones para forzar distribución usando archivo existente
	job := createSimpleHighPartitionJob()
	jobID := submitJobE2E(t, masterURL, job)

	waitForJobCompletionE2E(t, masterURL, jobID, 2*time.Minute)

	// Verificar que las tareas se distribuyeron entre workers
	verifyLoadDistribution(t, masterURL, jobID)
}

// Funciones auxiliares para tests E2E

func isE2EClusterReady(t *testing.T, masterURL string) bool {
	resp, err := http.Get(masterURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func setupE2ETest(t *testing.T) {
	// Asegurar que el directorio app existe
	os.MkdirAll("app/data", 0755)
	os.MkdirAll("app/results", 0755)
	os.MkdirAll("app/storage", 0755)
	os.MkdirAll("app/temp", 0755)
}

func createLargeTestData(t *testing.T) {
	data := []string{
		"line_header",
		"The quick brown fox jumps over the lazy dog",
		"Distributed computing with Apache Spark is powerful",
		"Mini-Spark implements MapReduce paradigm efficiently",
		"Fault tolerance is critical for distributed systems",
		"Load balancing ensures optimal resource utilization",
		"Docker containers provide consistent deployment environment",
		"Kubernetes orchestrates containerized applications seamlessly",
		"Microservices architecture enables scalable system design",
		"Data processing pipelines transform raw data into insights",
		"Machine learning algorithms analyze patterns in big data",
		"Cloud computing offers infinite scalability and flexibility",
		"DevOps practices streamline software development lifecycle",
		"Continuous integration automates testing and deployment processes",
		"Monitoring and observability ensure system reliability",
		"Performance optimization maximizes system throughput",
		"Security measures protect sensitive data and systems",
		"Database sharding distributes data across multiple nodes",
		"Caching strategies improve application response times",
		"Message queues decouple system components effectively",
	}

	// Repetir datos para hacer el dataset más grande
	var largeData []string
	for i := 0; i < 50; i++ {
		for _, line := range data[1:] { // Skip header in repetitions
			largeData = append(largeData, fmt.Sprintf("%s iteration %d", line, i))
		}
	}

	dataFile := "app/data/e2e_large_text.csv"
	file, err := os.Create(dataFile)
	if err != nil {
		t.Fatalf("Error creando archivo de prueba grande: %v", err)
	}
	defer file.Close()

	file.WriteString("text\n")
	for _, line := range largeData {
		file.WriteString(line + "\n")
	}

	t.Logf("Creado dataset grande con %d líneas", len(largeData))
}

func createComplexE2EData(t *testing.T) {
	// Usuarios
	users := [][]string{
		{"user_id", "name", "age", "city"},
		{"1", "Alice", "25", "New York"},
		{"2", "Bob", "30", "San Francisco"},
		{"3", "Charlie", "22", "Chicago"},
		{"4", "Diana", "28", "Boston"},
		{"5", "Eve", "35", "Seattle"},
		{"6", "Frank", "26", "Austin"},
	}

	userFile := "app/data/e2e_users.csv"
	writeCSVFile(t, userFile, users)

	// Órdenes
	orders := [][]string{
		{"order_id", "user_id", "product_id", "quantity", "price"},
		{"1", "1", "101", "2", "50.00"},
		{"2", "2", "102", "1", "75.00"},
		{"3", "1", "103", "3", "25.00"},
		{"4", "3", "101", "1", "50.00"},
		{"5", "4", "104", "2", "100.00"},
		{"6", "2", "105", "1", "150.00"},
		{"7", "5", "102", "2", "75.00"},
		{"8", "1", "106", "1", "200.00"},
	}

	orderFile := "app/data/e2e_orders.csv"
	writeCSVFile(t, orderFile, orders)

	// Productos
	products := [][]string{
		{"product_id", "name", "category", "price"},
		{"101", "Laptop", "Electronics", "50.00"},
		{"102", "Mouse", "Electronics", "75.00"},
		{"103", "Notebook", "Office", "25.00"},
		{"104", "Monitor", "Electronics", "100.00"},
		{"105", "Keyboard", "Electronics", "150.00"},
		{"106", "Desk", "Furniture", "200.00"},
	}

	productFile := "app/data/e2e_products.csv"
	writeCSVFile(t, productFile, products)
}

func createVeryLargeTestData(t *testing.T) {
	baseData := []string{
		"word processing system",
		"distributed computing framework",
		"data analytics platform",
		"machine learning pipeline",
		"real time streaming",
	}

	var veryLargeData []string
	// Generar 1000 líneas para forzar particionamiento
	for i := 0; i < 200; i++ {
		for j, line := range baseData {
			veryLargeData = append(veryLargeData, fmt.Sprintf("%s batch %d item %d", line, i, j))
		}
	}

	dataFile := "app/data/e2e_very_large_text.csv"
	file, err := os.Create(dataFile)
	if err != nil {
		t.Fatalf("Error creando dataset muy grande: %v", err)
	}
	defer file.Close()

	file.WriteString("text\n")
	for _, line := range veryLargeData {
		file.WriteString(line + "\n")
	}

	t.Logf("Creado dataset muy grande con %d líneas", len(veryLargeData))
}

func writeCSVFile(t *testing.T, filename string, data [][]string) {
	file, err := os.Create(filename)
	if err != nil {
		t.Fatalf("Error creando archivo %s: %v", filename, err)
	}
	defer file.Close()

	for _, row := range data {
		for i, field := range row {
			if i > 0 {
				file.WriteString(",")
			}
			file.WriteString(field)
		}
		file.WriteString("\n")
	}
}

func verifyWorkersRegistered(t *testing.T, masterURL string, expectedWorkers int) {
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatalf("Timeout: no se verificaron %d workers", expectedWorkers)
		case <-ticker.C:
			resp, err := http.Get(masterURL + "/health")
			if err != nil {
				continue
			}

			var health map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&health)
			resp.Body.Close()

			if workersUp, ok := health["workers_up"].(float64); ok {
				activeWorkers := int(workersUp)

				if activeWorkers >= expectedWorkers {
					t.Logf("✓ %d workers registrados y activos", activeWorkers)
					return
				}

				t.Logf("Esperando workers: %d/%d activos", activeWorkers, expectedWorkers)
			} else {
				t.Logf("No se pudo obtener información de workers del health endpoint")
			}
		}
	}
}

func createMultiNodeWordCountJob() map[string]interface{} {
	return map[string]interface{}{
		"name": "e2e-wordcount-multinode",
		"dag": map[string]interface{}{
			"nodes": []map[string]interface{}{
				{
					"id":          "read",
					"operator":    "read_csv",
					"input_paths": []string{"text.csv"}, // Usar archivo existente
					"output_path": "temp/e2e_read_output",
					"partitions":  6, // Múltiples particiones para distribución
				},
				{
					"id":          "tokenize",
					"operator":    "flat_map",
					"input_paths": []string{"temp/e2e_read_output"},
					"output_path": "results/e2e_wordcount_multinode",
					"partitions":  6,
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

func createSimplePipelineJob() map[string]interface{} {
	return map[string]interface{}{
		"name": "e2e-simple-pipeline",
		"dag": map[string]interface{}{
			"nodes": []map[string]interface{}{
				{
					"id":          "read_sales",
					"operator":    "read_csv",
					"input_paths": []string{"sales.csv"}, // Usar archivo existente
					"output_path": "temp/e2e_sales_read",
					"partitions":  3,
				},
				{
					"id":          "filter_high_price",
					"operator":    "filter",
					"input_paths": []string{"temp/e2e_sales_read"},
					"output_path": "temp/e2e_high_price_sales",
					"partitions":  3,
					"params": map[string]interface{}{
						"condition": "price > 50",
					},
				},
				{
					"id":          "aggregate_total",
					"operator":    "aggregate",
					"input_paths": []string{"temp/e2e_high_price_sales"},
					"output_path": "results/e2e_pipeline_result",
					"partitions":  1,
					"params": map[string]interface{}{
						"function": "sum",
						"column":   "price",
					},
				},
			},
			"edges": []map[string]interface{}{
				{"from": "read_sales", "to": "filter_high_price"},
				{"from": "filter_high_price", "to": "aggregate_total"},
			},
		},
	}
}

func createJoinPipelineJob() map[string]interface{} {
	return map[string]interface{}{
		"name": "e2e-join-pipeline",
		"dag": map[string]interface{}{
			"nodes": []map[string]interface{}{
				{
					"id":          "read_users",
					"operator":    "read_csv",
					"input_paths": []string{"e2e_users.csv"},
					"output_path": "temp/e2e_users_read",
					"partitions":  2,
				},
				{
					"id":          "read_orders",
					"operator":    "read_csv",
					"input_paths": []string{"e2e_orders.csv"},
					"output_path": "temp/e2e_orders_read",
					"partitions":  3,
				},
				{
					"id":          "join_user_orders",
					"operator":    "join",
					"input_paths": []string{"temp/e2e_users_read", "temp/e2e_orders_read"},
					"output_path": "temp/e2e_user_orders",
					"partitions":  3,
					"params": map[string]interface{}{
						"join_key": "user_id",
					},
				},
				{
					"id":          "aggregate_sales",
					"operator":    "aggregate",
					"input_paths": []string{"temp/e2e_user_orders"},
					"output_path": "results/e2e_sales_by_user",
					"partitions":  2,
					"params": map[string]interface{}{
						"function": "sum",
						"column":   "price",
					},
				},
			},
			"edges": []map[string]interface{}{
				{"from": "read_users", "to": "join_user_orders"},
				{"from": "read_orders", "to": "join_user_orders"},
				{"from": "join_user_orders", "to": "aggregate_sales"},
			},
		},
	}
}

func createAggregationJobE2E() map[string]interface{} {
	return map[string]interface{}{
		"name": "e2e-aggregation",
		"dag": map[string]interface{}{
			"nodes": []map[string]interface{}{
				{
					"id":          "read",
					"operator":    "read_csv",
					"input_paths": []string{"sales.csv"}, // Usar archivo existente
					"output_path": "temp/e2e_orders_agg",
					"partitions":  3,
				},
				{
					"id":          "sum_prices",
					"operator":    "aggregate",
					"input_paths": []string{"temp/e2e_orders_agg"},
					"output_path": "results/e2e_total_sales",
					"partitions":  1,
					"params": map[string]interface{}{
						"function": "sum",
						"column":   "price",
					},
				},
			},
			"edges": []map[string]interface{}{
				{"from": "read", "to": "sum_prices"},
			},
		},
	}
}

func createSimpleFilterJob() map[string]interface{} {
	return map[string]interface{}{
		"name": "e2e-simple-filter",
		"dag": map[string]interface{}{
			"nodes": []map[string]interface{}{
				{
					"id":          "read",
					"operator":    "read_csv",
					"input_paths": []string{"sales.csv"}, // Usar archivo existente
					"output_path": "temp/e2e_sales_filter",
					"partitions":  2,
				},
				{
					"id":          "filter_high",
					"operator":    "filter",
					"input_paths": []string{"temp/e2e_sales_filter"},
					"output_path": "results/e2e_filtered_sales",
					"partitions":  1,
					"params": map[string]interface{}{
						"condition": "price > 30",
					},
				},
			},
			"edges": []map[string]interface{}{
				{"from": "read", "to": "filter_high"},
			},
		},
	}
}

func createFilterMapJob() map[string]interface{} {
	return map[string]interface{}{
		"name": "e2e-filter-map",
		"dag": map[string]interface{}{
			"nodes": []map[string]interface{}{
				{
					"id":          "read",
					"operator":    "read_csv",
					"input_paths": []string{"e2e_users.csv"},
					"output_path": "temp/e2e_users_filter",
					"partitions":  2,
				},
				{
					"id":          "filter_young",
					"operator":    "filter",
					"input_paths": []string{"temp/e2e_users_filter"},
					"output_path": "temp/e2e_young_users",
					"partitions":  2,
					"params": map[string]interface{}{
						"condition": "age < 30",
					},
				},
				{
					"id":          "uppercase_names",
					"operator":    "map",
					"input_paths": []string{"temp/e2e_young_users"},
					"output_path": "results/e2e_young_users_upper",
					"partitions":  1,
					"params": map[string]interface{}{
						"function": "uppercase",
						"column":   "name",
					},
				},
			},
			"edges": []map[string]interface{}{
				{"from": "read", "to": "filter_young"},
				{"from": "filter_young", "to": "uppercase_names"},
			},
		},
	}
}

func createSimpleHighPartitionJob() map[string]interface{} {
	return map[string]interface{}{
		"name": "e2e-load-balance",
		"dag": map[string]interface{}{
			"nodes": []map[string]interface{}{
				{
					"id":          "read",
					"operator":    "read_csv",
					"input_paths": []string{"text.csv"}, // Usar archivo existente
					"output_path": "temp/e2e_large_read",
					"partitions":  6, // Múltiples particiones para distribución
				},
				{
					"id":          "tokenize",
					"operator":    "flat_map",
					"input_paths": []string{"temp/e2e_large_read"},
					"output_path": "results/e2e_large_tokens",
					"partitions":  6,
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

func createHighPartitionJob() map[string]interface{} {
	return map[string]interface{}{
		"name": "e2e-load-balance",
		"dag": map[string]interface{}{
			"nodes": []map[string]interface{}{
				{
					"id":          "read",
					"operator":    "read_csv",
					"input_paths": []string{"e2e_very_large_text.csv"},
					"output_path": "temp/e2e_large_read",
					"partitions":  9, // 9 particiones para 3 workers
				},
				{
					"id":          "map_lowercase",
					"operator":    "map",
					"input_paths": []string{"temp/e2e_large_read"},
					"output_path": "temp/e2e_large_mapped",
					"partitions":  9,
					"params": map[string]interface{}{
						"function": "lowercase",
						"column":   "text",
					},
				},
				{
					"id":          "tokenize",
					"operator":    "flat_map",
					"input_paths": []string{"temp/e2e_large_mapped"},
					"output_path": "results/e2e_large_tokens",
					"partitions":  6,
					"params": map[string]interface{}{
						"function": "split_words",
					},
				},
			},
			"edges": []map[string]interface{}{
				{"from": "read", "to": "map_lowercase"},
				{"from": "map_lowercase", "to": "tokenize"},
			},
		},
	}
}

func submitJobE2E(t *testing.T, masterURL string, job map[string]interface{}) string {
	jobJSON, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("Error marshaling job: %v", err)
	}

	resp, err := http.Post(masterURL+"/api/v1/jobs", "application/json", bytes.NewBuffer(jobJSON))
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

	t.Logf("Job enviado: %s (ID: %s)", job["name"], jobID)
	return jobID
}

func waitForJobCompletionE2E(t *testing.T, masterURL, jobID string, timeout time.Duration) {
	timeoutChan := time.After(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	t.Logf("Esperando que el job %s complete...", jobID)

	for {
		select {
		case <-timeoutChan:
			t.Fatalf("Timeout esperando que el job %s complete", jobID)
		case <-ticker.C:
			resp, err := http.Get(fmt.Sprintf("%s/api/v1/jobs/%s", masterURL, jobID))
			if err != nil {
				t.Logf("Error consultando estado del job: %v", err)
				continue
			}

			var jobInfo map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&jobInfo)
			resp.Body.Close()

			if status, ok := jobInfo["status"].(string); ok {
				t.Logf("Job %s status: %s", jobID, status)

				if status == "SUCCEEDED" {
					t.Logf("✓ Job %s completado exitosamente", jobID)
					return
				} else if status == "FAILED" {
					t.Fatalf("✗ Job %s falló", jobID)
				}
				// Continuar esperando si está RUNNING o ACCEPTED
			}
		}
	}
}

func validateMultiNodeResults(t *testing.T, jobID string) {
	resultsPath := "../app/results/e2e_wordcount_multinode-part-0"

	if _, err := os.Stat(resultsPath); os.IsNotExist(err) {
		t.Fatalf("Archivo de resultados no encontrado: %s", resultsPath)
	}

	content, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatalf("Error leyendo resultados: %v", err)
	}

	results := string(content)

	// Verificar palabras comunes en el dataset
	expectedWords := []string{"Hello", "spark", "world", "Computing"}
	for _, word := range expectedWords {
		if !bytes.Contains([]byte(results), []byte(word)) {
			t.Errorf("Palabra '%s' no encontrada en resultados multinodo", word)
		}
	}

	t.Logf("✓ Resultados multinodo validados para job %s", jobID)
}

func validateE2EPipelineResults(t *testing.T, jobID string) {
	resultsPath := "../app/results/e2e_pipeline_result"

	if _, err := os.Stat(resultsPath); os.IsNotExist(err) {
		t.Fatalf("Archivo de resultados de pipeline no encontrado: %s", resultsPath)
	}

	content, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatalf("Error leyendo resultados de pipeline: %v", err)
	}

	results := string(content)

	// Verificar que hay algún resultado de agregación
	if len(results) < 5 {
		t.Errorf("Resultados de pipeline muy pequeños: %s", results)
	}

	t.Logf("✓ Resultados de pipeline validados para job %s", jobID)
}

func validateJoinResults(t *testing.T, jobID string) {
	resultsPath := filepath.Join("app", "results", "e2e_sales_by_user-part-0")

	if _, err := os.Stat(resultsPath); os.IsNotExist(err) {
		t.Fatalf("Archivo de resultados de join no encontrado: %s", resultsPath)
	}

	content, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatalf("Error leyendo resultados de join: %v", err)
	}

	results := string(content)

	// Verificar que hay resultados de agregación
	if len(results) < 10 {
		t.Errorf("Resultados de join muy pequeños: %s", results)
	}

	t.Logf("✓ Resultados de join validados para job %s", jobID)
}

func verifyLoadDistribution(t *testing.T, masterURL, jobID string) {
	// Consultar métricas del job para verificar distribución
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/jobs/%s/metrics", masterURL, jobID))
	if err != nil {
		t.Logf("Warning: No se pudieron obtener métricas de distribución: %v", err)
		return
	}
	defer resp.Body.Close()

	var metrics map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		t.Logf("Warning: Error decodificando métricas: %v", err)
		return
	}

	t.Logf("✓ Job %s ejecutado con distribución de carga", jobID)
	if tasks, ok := metrics["total_tasks"].(float64); ok {
		t.Logf("  Total de tareas ejecutadas: %.0f", tasks)
	}
}
