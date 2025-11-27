package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/juandjimenezjdjv/Proy_2_SO/common"
)

// Colores ANSI
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"
)

// Config del cliente
type ClientConfig struct {
	MasterURL string
}

func main() {
	printBanner()

	// Parse argumentos
	var (
		masterURL = flag.String("master", "http://localhost:8080", "URL del Master")
		command   = flag.String("cmd", "", "Comando: submit, status, list, health")
		jobFile   = flag.String("job", "", "Archivo JSON con definición del job")
		jobID     = flag.String("id", "", "ID del job para consultar estado")
		watch     = flag.Bool("watch", false, "Monitorear job hasta completarse")
	)
	flag.Parse()

	if *command == "" {
		printUsage()
		os.Exit(1)
	}

	client := &Client{
		masterURL: *masterURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	var err error
	switch *command {
	case "submit":
		err = client.submitJob(*jobFile, *watch)
	case "status":
		err = client.getJobStatus(*jobID)
	case "list":
		err = client.listJobs()
	case "health":
		err = client.checkHealth()
	default:
		printError(fmt.Sprintf("Comando desconocido: %s", *command))
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}
}

// printBanner muestra el banner del CLI
func printBanner() {
	banner := `
  ███╗   ███╗██╗███╗   ██╗██╗    ███████╗██████╗  █████╗ ██████╗ ██╗  ██╗
  ████╗ ████║██║████╗  ██║██║    ██╔════╝██╔══██╗██╔══██╗██╔══██╗██║ ██╔╝
  ██╔████╔██║██║██╔██╗ ██║██║    ███████╗██████╔╝███████║██████╔╝█████╔╝ 
  ██║╚██╔╝██║██║██║╚██╗██║██║    ╚════██║██╔═══╝ ██╔══██║██╔══██╗██╔═██╗ 
  ██║ ╚═╝ ██║██║██║ ╚████║██║    ███████║██║     ██║  ██║██║  ██║██║  ██╗
  ╚═╝     ╚═╝╚═╝╚═╝  ╚═══╝╚═╝    ╚══════╝╚═╝     ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝
`
	fmt.Printf("%s%s%s", ColorCyan, banner, ColorReset)
	fmt.Printf("%s         Distributed Batch Processing System - CLI Client%s\n\n", ColorDim, ColorReset)
}

// printError imprime un mensaje de error con estilo
func printError(msg string) {
	fmt.Printf("\n%s✗ ERROR:%s %s\n\n", ColorRed+ColorBold, ColorReset, msg)
}

// printSuccess imprime un mensaje de éxito con estilo
func printSuccess(msg string) {
	fmt.Printf("%s✓%s %s\n", ColorGreen+ColorBold, ColorReset, msg)
}

// printInfo imprime información con estilo
func printInfo(msg string) {
	fmt.Printf("%sℹ%s  %s\n", ColorBlue+ColorBold, ColorReset, msg)
}

// printWarning imprime una advertencia con estilo
func printWarning(msg string) {
	fmt.Printf("%s⚠%s  %s\n", ColorYellow+ColorBold, ColorReset, msg)
}

// printSeparator imprime un separador visual
func printSeparator() {
	fmt.Printf("%s%s%s\n", ColorDim, strings.Repeat("─", 70), ColorReset)
}

// printHeader imprime un encabezado de sección
func printHeader(title string) {
	fmt.Printf("\n%s╔═══ %s ═══╗%s\n", ColorCyan+ColorBold, strings.ToUpper(title), ColorReset)
}

// Client maneja comunicación con el Master
type Client struct {
	masterURL  string
	httpClient *http.Client
}

// submitJob envía un job al Master
func (c *Client) submitJob(jobFile string, watch bool) error {
	if jobFile == "" {
		return fmt.Errorf("debe especificar un archivo de job con -job")
	}

	printHeader("Submit Job")

	// Leer archivo de job
	printInfo(fmt.Sprintf("Leyendo archivo: %s", jobFile))
	data, err := os.ReadFile(jobFile)
	if err != nil {
		return fmt.Errorf("error leyendo archivo: %w", err)
	}

	// Parsear JSON
	var job common.Job
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Errorf("error parseando JSON: %w", err)
	}

	printInfo(fmt.Sprintf("Job ID: %s%s%s", ColorYellow, job.ID, ColorReset))
	printInfo(fmt.Sprintf("Nodos DAG: %s%d%s", ColorCyan, len(job.DAG.Nodes), ColorReset))
	printSeparator()

	// Enviar al Master
	fmt.Printf("%s⚡ Enviando al Master...%s ", ColorYellow, ColorReset)
	resp, err := c.httpClient.Post(
		c.masterURL+"/api/v1/jobs",
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		fmt.Println()
		return fmt.Errorf("error enviando job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		fmt.Println()
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error del servidor (%d): %s", resp.StatusCode, string(body))
	}

	fmt.Printf("%sDone!%s\n\n", ColorGreen, ColorReset)
	printSuccess("Job enviado exitosamente")
	printInfo(fmt.Sprintf("Job ID: %s%s%s", ColorPurple+ColorBold, job.ID, ColorReset))

	// Monitorear si se solicitó
	if watch {
		fmt.Println()
		printSeparator()
		printInfo("Iniciando monitoreo en tiempo real...")
		return c.watchJob(job.ID)
	}

	fmt.Printf("\n%sTip:%s Para monitorear: %s./client.exe -cmd status -id %s%s\n\n", ColorDim, ColorReset, ColorCyan, job.ID, ColorReset)
	return nil
}

// getJobStatus consulta el estado de un job
func (c *Client) getJobStatus(jobID string) error {
	if jobID == "" {
		return fmt.Errorf("debe especificar un job ID con -id")
	}

	printHeader("Job Status")

	resp, err := c.httpClient.Get(c.masterURL + "/api/v1/jobs/" + jobID)
	if err != nil {
		return fmt.Errorf("error consultando estado: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("job no encontrado: %s", jobID)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error del servidor: %d", resp.StatusCode)
	}

	var job common.Job
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return fmt.Errorf("error parseando respuesta: %w", err)
	}

	// Mostrar información del job
	c.printJobInfo(&job)
	return nil
}

// listJobs lista todos los jobs
func (c *Client) listJobs() error {
	printHeader("Job List")

	resp, err := c.httpClient.Get(c.masterURL + "/api/v1/jobs")
	if err != nil {
		return fmt.Errorf("error listando jobs: %w", err)
	}
	defer resp.Body.Close()

	var jobs []common.Job
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		return fmt.Errorf("error parseando respuesta: %w", err)
	}

	if len(jobs) == 0 {
		printInfo("No hay jobs en el sistema")
		fmt.Printf("\n%sTip:%s Use %s./client -cmd submit -job <file>%s para enviar un job\n\n", ColorDim, ColorReset, ColorCyan, ColorReset)
		return nil
	}

	fmt.Printf("\n%sTotal: %d jobs%s\n\n", ColorBold, len(jobs), ColorReset)
	printSeparator()

	// Tabla con formato
	fmt.Printf("%s%-30s %-15s %-8s %-20s%s\n", ColorBold, "JOB ID", "STATUS", "NODES", "SUBMITTED", ColorReset)
	printSeparator()

	for _, job := range jobs {
		statusColor := getStatusColor(string(job.Status))
		timeStr := "-"
		if !job.SubmittedAt.IsZero() {
			timeStr = job.SubmittedAt.Format("15:04:05 02/01")
		}
		fmt.Printf("%-30s %s%-15s%s %-8d %-20s\n",
			truncateString(job.ID, 28),
			statusColor,
			string(job.Status),
			ColorReset,
			len(job.DAG.Nodes),
			timeStr,
		)
	}

	printSeparator()
	fmt.Println()
	return nil
}

// checkHealth verifica la salud del cluster
func (c *Client) checkHealth() error {
	printHeader("Cluster Health")

	resp, err := c.httpClient.Get(c.masterURL + "/health")
	if err != nil {
		return fmt.Errorf("error verificando salud: %w", err)
	}
	defer resp.Body.Close()

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return fmt.Errorf("error parseando respuesta: %w", err)
	}

	// Status principal
	status, _ := health["status"].(string)
	if status == "healthy" {
		fmt.Printf("\n  %s●%s Status: %s%s%s\n", ColorGreen+ColorBold, ColorReset, ColorGreen, status, ColorReset)
	} else {
		fmt.Printf("\n  %s●%s Status: %s%s%s\n", ColorRed+ColorBold, ColorReset, ColorRed, status, ColorReset)
	}

	// Master Info
	if masterID, ok := health["master_id"].(string); ok {
		fmt.Printf("  %s▸%s Master: %s%s%s\n", ColorBlue, ColorReset, ColorCyan, masterID, ColorReset)
	}

	// Workers
	workersTotal, _ := health["workers_total"].(float64)
	workersUp, _ := health["workers_up"].(float64)
	workersColor := ColorGreen
	if workersUp < workersTotal {
		workersColor = ColorYellow
	}
	if workersUp == 0 {
		workersColor = ColorRed
	}
	fmt.Printf("  %s▸%s Workers: %s%.0f/%.0f%s online\n", ColorBlue, ColorReset, workersColor, workersUp, workersTotal, ColorReset)

	// Visualizar workers
	fmt.Printf("\n  ")
	for i := 0; i < int(workersTotal); i++ {
		if i < int(workersUp) {
			fmt.Printf("%s[█]%s ", ColorGreen, ColorReset)
		} else {
			fmt.Printf("%s[░]%s ", ColorRed, ColorReset)
		}
	}
	fmt.Println()

	// Timestamp
	if timestamp, ok := health["timestamp"].(string); ok {
		t, _ := time.Parse(time.RFC3339, timestamp)
		fmt.Printf("\n  %sLast check: %s%s\n", ColorDim, t.Format("15:04:05 02/01/2006"), ColorReset)
	}

	fmt.Println()
	return nil
}

// watchJob monitorea un job hasta que termine
func (c *Client) watchJob(jobID string) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastStatus := common.JobStatus("")
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinnerIdx := 0

	for {
		select {
		case <-ticker.C:
			resp, err := c.httpClient.Get(c.masterURL + "/api/v1/jobs/" + jobID)
			if err != nil {
				printWarning(fmt.Sprintf("Error consultando estado: %v", err))
				continue
			}

			var job common.Job
			json.NewDecoder(resp.Body).Decode(&job)
			resp.Body.Close()

			// Mostrar update si cambió el estado
			if job.Status != lastStatus {
				statusColor := getStatusColor(string(job.Status))
				fmt.Printf("\n  %s[%s]%s Status: %s%s%s\n",
					ColorCyan,
					time.Now().Format("15:04:05"),
					ColorReset,
					statusColor,
					job.Status,
					ColorReset,
				)
				lastStatus = job.Status

				// Mostrar progreso de tareas
				if len(job.Tasks) > 0 {
					completed := 0
					for _, task := range job.Tasks {
						if task.Status == common.TaskCompleted {
							completed++
						}
					}
					progress := float64(completed) / float64(len(job.Tasks)) * 100
					progressBar := createProgressBar(int(progress), 30)
					fmt.Printf("  %s %s %.0f%%%s (%d/%d tasks)\n",
						progressBar,
						ColorCyan,
						progress,
						ColorReset,
						completed,
						len(job.Tasks),
					)
				}
			} else {
				// Animación de espera
				fmt.Printf("\r  %s%s%s Waiting...",
					ColorCyan,
					spinner[spinnerIdx],
					ColorReset,
				)
				spinnerIdx = (spinnerIdx + 1) % len(spinner)
			}

			// Terminar si el job finalizó
			if job.Status == common.JobCompleted {
				fmt.Println()
				printSeparator()
				printSuccess("🎉 Job completado exitosamente")
				c.printJobInfo(&job)
				return nil
			}
			if job.Status == common.JobFailed {
				fmt.Println()
				printSeparator()
				printError("Job falló")
				c.printJobInfo(&job)
				return fmt.Errorf("job falló")
			}
		}
	}
}

// printJobInfo imprime información detallada de un job
func (c *Client) printJobInfo(job *common.Job) {
	fmt.Println()
	printSeparator()

	// Encabezado con ID
	fmt.Printf("\n  %s▸ Job:%s %s%s%s\n", ColorCyan+ColorBold, ColorReset, ColorPurple, job.ID, ColorReset)

	// Estado con color
	statusColor := getStatusColor(string(job.Status))
	fmt.Printf("  %s▸ Status:%s %s%s%s\n", ColorCyan+ColorBold, ColorReset, statusColor, job.Status, ColorReset)

	// Info del DAG
	fmt.Printf("  %s▸ DAG:%s %d nodes, %d edges\n", ColorCyan+ColorBold, ColorReset, len(job.DAG.Nodes), len(job.DAG.Edges))

	// Timestamps
	fmt.Println()
	if !job.SubmittedAt.IsZero() {
		fmt.Printf("  %s⏱  Submitted:%s %s\n", ColorDim, ColorReset, job.SubmittedAt.Format("15:04:05 02/01/2006"))
	}
	if !job.StartedAt.IsZero() {
		fmt.Printf("  %s▶  Started:%s   %s\n", ColorDim, ColorReset, job.StartedAt.Format("15:04:05 02/01/2006"))
	}
	if !job.CompletedAt.IsZero() {
		duration := job.CompletedAt.Sub(job.StartedAt)
		fmt.Printf("  %s✓  Completed:%s %s %s(duration: %s)%s\n",
			ColorDim, ColorReset,
			job.CompletedAt.Format("15:04:05 02/01/2006"),
			ColorGreen,
			duration.Round(time.Millisecond),
			ColorReset,
		)
	}

	// Mostrar tareas
	if len(job.Tasks) > 0 {
		fmt.Println()
		fmt.Printf("  %s▸ Tasks:%s %d total\n", ColorCyan+ColorBold, ColorReset, len(job.Tasks))
		printSeparator()

		statusCount := make(map[common.TaskStatus]int)
		for _, task := range job.Tasks {
			statusCount[task.Status]++
		}

		// Mostrar conteo con barras
		for status, count := range statusCount {
			percentage := float64(count) / float64(len(job.Tasks)) * 100
			bar := createProgressBar(int(percentage), 20)
			statusColor := getStatusColor(string(status))
			fmt.Printf("  %s%-12s%s %s %3d (%.0f%%)\n",
				statusColor,
				status,
				ColorReset,
				bar,
				count,
				percentage,
			)
		}
	}

	printSeparator()
	fmt.Println()
}

// printUsage muestra el uso del cliente
func printUsage() {
	fmt.Printf("\n%s╔════════════════════════════════════════════════════════════════╗%s\n", ColorCyan, ColorReset)
	fmt.Printf("%s║%s           %sMINI-SPARK CLI - Command Reference%s               %s║%s\n", ColorCyan, ColorReset, ColorBold, ColorReset, ColorCyan, ColorReset)
	fmt.Printf("%s╚════════════════════════════════════════════════════════════════╝%s\n\n", ColorCyan, ColorReset)

	fmt.Printf("%sUSAGE:%s\n", ColorYellow+ColorBold, ColorReset)
	fmt.Printf("  ./client -cmd <command> [options]\n\n")

	fmt.Printf("%sCOMMANDS:%s\n", ColorYellow+ColorBold, ColorReset)
	fmt.Printf("  %ssubmit%s    📤 Submit a job to the cluster\n", ColorGreen, ColorReset)
	fmt.Printf("  %sstatus%s    📊 Get job status and details\n", ColorGreen, ColorReset)
	fmt.Printf("  %slist%s      📋 List all jobs in the system\n", ColorGreen, ColorReset)
	fmt.Printf("  %shealth%s    🏥 Check cluster health\n\n", ColorGreen, ColorReset)

	fmt.Printf("%sOPTIONS:%s\n", ColorYellow+ColorBold, ColorReset)
	fmt.Printf("  %s-master%s <url>     Master URL (default: http://localhost:8080)\n", ColorCyan, ColorReset)
	fmt.Printf("  %s-job%s <file>       Job definition file (JSON)\n", ColorCyan, ColorReset)
	fmt.Printf("  %s-id%s <job-id>      Job ID for status query\n", ColorCyan, ColorReset)
	fmt.Printf("  %s-watch%s            Monitor job until completion\n\n", ColorCyan, ColorReset)

	fmt.Printf("%sEXAMPLES:%s\n", ColorYellow+ColorBold, ColorReset)
	fmt.Printf("  %s# Submit a job%s\n", ColorDim, ColorReset)
	fmt.Printf("  ./client -cmd submit -job job.json\n\n")

	fmt.Printf("  %s# Submit and watch%s\n", ColorDim, ColorReset)
	fmt.Printf("  ./client -cmd submit -job job.json -watch\n\n")

	fmt.Printf("  %s# Check job status%s\n", ColorDim, ColorReset)
	fmt.Printf("  ./client -cmd status -id example-job\n\n")

	fmt.Printf("  %s# List all jobs%s\n", ColorDim, ColorReset)
	fmt.Printf("  ./client -cmd list\n\n")

	fmt.Printf("  %s# Check cluster health%s\n", ColorDim, ColorReset)
	fmt.Printf("  ./client -cmd health\n\n")
}

// Helper functions

// getStatusColor retorna el color apropiado según el estado
func getStatusColor(status string) string {
	statusUpper := strings.ToUpper(status)
	switch {
	case strings.Contains(statusUpper, "COMPLETED") || strings.Contains(statusUpper, "SUCCEEDED"):
		return ColorGreen + ColorBold
	case strings.Contains(statusUpper, "RUNNING"):
		return ColorYellow + ColorBold
	case strings.Contains(statusUpper, "FAILED"):
		return ColorRed + ColorBold
	case strings.Contains(statusUpper, "PENDING") || strings.Contains(statusUpper, "ACCEPTED"):
		return ColorBlue + ColorBold
	default:
		return ColorWhite
	}
}

// createProgressBar crea una barra de progreso visual
func createProgressBar(percentage, width int) string {
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 100 {
		percentage = 100
	}

	filled := (percentage * width) / 100
	empty := width - filled

	bar := ColorGreen + strings.Repeat("█", filled) + ColorReset
	bar += ColorDim + strings.Repeat("░", empty) + ColorReset

	return "[" + bar + "]"
}

// truncateString trunca un string a una longitud máxima
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
