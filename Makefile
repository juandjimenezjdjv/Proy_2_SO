# Makefile para Mini-Spark
# Automatización de build, despliegue y testing

.PHONY: help build build-master build-worker build-client up down restart logs logs-master logs-workers status clean test demo

# Colores para output
GREEN  := \033[0;32m
YELLOW := \033[0;33m
RED    := \033[0;31m
NC     := \033[0m # No Color

##@ General

help: ## Muestra esta ayuda
	@echo "$(GREEN)Mini-Spark - Sistema de Procesamiento Distribuido$(NC)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "Uso:\n  make $(YELLOW)<target>$(NC)\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(GREEN)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build

build: ## Compila todos los componentes localmente
	@echo "$(GREEN)Compilando Master...$(NC)"
	@cd master && go build -o master .
	@echo "$(GREEN)Compilando Worker...$(NC)"
	@cd worker && go build -o worker .
	@echo "$(GREEN)✓ Compilación exitosa$(NC)"

build-master: ## Compila solo el master
	@echo "$(GREEN)Compilando Master...$(NC)"
	@cd master && go build -o master .
	@echo "$(GREEN)✓ Master compilado$(NC)"

build-worker: ## Compila solo el worker
	@echo "$(GREEN)Compilando Worker...$(NC)"
	@cd worker && go build -o worker .
	@echo "$(GREEN)✓ Worker compilado$(NC)"

build-client: ## Compila el cliente CLI (cuando esté implementado)
	@echo "$(YELLOW)Cliente CLI aún no implementado$(NC)"

##@ Docker

docker-build: ## Construye las imágenes Docker sin cache
	@echo "$(GREEN)Construyendo imágenes Docker...$(NC)"
	@docker compose build --no-cache
	@echo "$(GREEN)✓ Imágenes construidas$(NC)"

up: ## Inicia el cluster (1 master + 3 workers)
	@echo "$(GREEN)Iniciando cluster Mini-Spark...$(NC)"
	@mkdir -p app/data app/results app/storage app/temp
	@docker compose up -d --build
	@echo "$(GREEN)✓ Cluster iniciado$(NC)"
	@echo "$(YELLOW)Master API: http://localhost:8080$(NC)"
	@echo "$(YELLOW)Health check: http://localhost:8080/health$(NC)"
	@sleep 5
	@make status

down: ## Detiene y elimina el cluster
	@echo "$(RED)Deteniendo cluster...$(NC)"
	@docker compose down
	@echo "$(GREEN)✓ Cluster detenido$(NC)"

restart: down up ## Reinicia el cluster completo

##@ Logs y Monitoreo

logs: ## Muestra logs de todos los contenedores
	@docker compose logs -f

logs-master: ## Muestra logs solo del master
	@docker compose logs -f master

logs-workers: ## Muestra logs de todos los workers
	@docker compose logs -f worker1 worker2 worker3

status: ## Muestra el estado del cluster
	@echo "$(GREEN)Estado del Cluster:$(NC)"
	@docker compose ps
	@echo ""
	@echo "$(YELLOW)Health Status:$(NC)"
	@curl -s http://localhost:8080/health 2>/dev/null || echo "Master no disponible"

##@ Testing

test: ## Ejecuta pruebas unitarias
	@echo "$(GREEN)Ejecutando pruebas unitarias...$(NC)"
	@go test ./... -v -race
	@echo "$(GREEN)✓ Pruebas completadas$(NC)"

test-coverage: ## Ejecuta pruebas con reporte de cobertura
	@echo "$(GREEN)Ejecutando pruebas con cobertura...$(NC)"
	@go test ./... -v -race -coverprofile=coverage.out -covermode=atomic
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | grep total | awk '{print "$(GREEN)Cobertura total: " $$3 "$(NC)"}'
	@echo "$(YELLOW)Reporte HTML generado: coverage.html$(NC)"

test-verbose: ## Ejecuta pruebas con output detallado
	@echo "$(GREEN)Ejecutando pruebas detalladas...$(NC)"
	@go test ./... -v -race -count=1

test-unit: ## Ejecuta solo pruebas unitarias (sin integration/e2e)
	@echo "$(GREEN)Ejecutando pruebas unitarias...$(NC)"
	@go test -v -race ./common ./master ./worker
	@echo "$(GREEN)✓ Pruebas unitarias completadas$(NC)"

test-integration: ## Ejecuta pruebas de integración (nodo único)
	@echo "$(GREEN)Ejecutando pruebas de integración...$(NC)"
	@go test -v -race ./tests -run TestIntegration -timeout 10m
	@echo "$(GREEN)✓ Pruebas de integración completadas$(NC)"

test-e2e: up ## Ejecuta pruebas end-to-end (multinodo con Docker)
	@echo "$(GREEN)Ejecutando pruebas end-to-end...$(NC)"
	@go test -v -race ./tests -run TestE2E -timeout 15m
	@echo "$(GREEN)✓ Pruebas end-to-end completadas$(NC)"

test-all: test-unit test-integration test-e2e ## Ejecuta todos los tipos de pruebas
	@echo "$(GREEN)✓ Todas las pruebas completadas$(NC)"

test-quick: ## Ejecuta pruebas rápidas (solo unitarias)
	@echo "$(GREEN)Ejecutando pruebas rápidas...$(NC)"
	@go test -v -short ./...
	@echo "$(GREEN)✓ Pruebas rápidas completadas$(NC)"

##@ Limpieza

clean: down ## Limpia binarios compilados y datos temporales
	@echo "$(RED)Limpiando archivos temporales...$(NC)"
	@rm -f master/master worker/worker client/client
	@rm -rf app/results/* app/storage/* app/temp/*
	@echo "$(GREEN)✓ Limpieza completada$(NC)"

clean-all: clean ## Limpieza completa incluyendo imágenes Docker
	@echo "$(RED)Eliminando imágenes Docker...$(NC)"
	@docker compose down -v --rmi all
	@echo "$(GREEN)✓ Limpieza completa$(NC)"

##@ Demo

demo: up ## Inicia demo interactivo del sistema
	@echo "$(GREEN)=== Demo Mini-Spark ===$(NC)"
	@sleep 5
	@echo ""
	@echo "$(YELLOW)1. Health Check:$(NC)"
	@curl -s http://localhost:8080/health | head -c 100
	@echo ""
	@echo ""
	@echo "$(YELLOW)2. Workers Registrados:$(NC)"
	@curl -s http://localhost:8080/api/v1/workers | head -c 300
	@echo ""
	@echo ""
	@echo "$(YELLOW)3. Lista de Jobs:$(NC)"
	@curl -s http://localhost:8080/api/v1/jobs | head -c 100
	@echo ""
	@echo ""
	@echo "$(GREEN)✓ Demo completado$(NC)"
	@echo "$(YELLOW)Ver logs con: make logs$(NC)"

##@ Desarrollo

dev-master: ## Ejecuta el master en modo desarrollo (local)
	@echo "$(GREEN)Iniciando Master en modo desarrollo...$(NC)"
	@cd master && go run main.go

dev-worker: ## Ejecuta un worker en modo desarrollo (local)
	@echo "$(GREEN)Iniciando Worker en modo desarrollo...$(NC)"
	@cd worker && go run main.go

fmt: ## Formatea el código Go
	@echo "$(GREEN)Formateando código...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)✓ Código formateado$(NC)"

lint: ## Ejecuta linter en el código
	@echo "$(GREEN)Ejecutando linter...$(NC)"
	@go vet ./...
	@echo "$(GREEN)✓ Linting completado$(NC)"
