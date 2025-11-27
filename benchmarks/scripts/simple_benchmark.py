#!/usr/bin/env python3
"""
Script simplificado de benchmark para Windows
"""

import json
import time
import sys
import subprocess
import os
from datetime import datetime

# Configuración
MASTER_URL = "http://localhost:8080"
BENCHMARK_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DATA_DIR = os.path.join(BENCHMARK_DIR, "data")
RESULTS_DIR = os.path.join(BENCHMARK_DIR, "results")
JOBS_DIR = os.path.join(BENCHMARK_DIR, "jobs")

def print_header(text):
    print("\n" + "=" * 70)
    print(f"  {text}")
    print("=" * 70 + "\n")

def check_cluster():
    """Verifica que el cluster esté activo"""
    print("[INFO] Verificando estado del cluster...")
    try:
        result = subprocess.run(
            ["curl", "-s", f"{MASTER_URL}/health"],
            capture_output=True,
            text=True,
            timeout=5
        )
        if result.returncode == 0:
            print(f"[INFO] ✓ Cluster activo en {MASTER_URL}")
            return True
        else:
            print(f"[ERROR] Cluster no responde en {MASTER_URL}")
            print("[INFO] Inicia el cluster con: docker compose up -d")
            return False
    except Exception as e:
        print(f"[ERROR] No se pudo conectar al cluster: {e}")
        return False

def run_job(job_file, job_name, description):
    """Ejecuta un job y mide el tiempo"""
    print(f"\n[INFO] Ejecutando: {job_name}")
    print(f"[INFO] Descripción: {description}")
    
    start_time = time.time()
    
    # Leer el job
    with open(job_file, 'r') as f:
        job_data = f.read()
    
    # Enviar job
    try:
        result = subprocess.run(
            ["curl", "-s", "-X", "POST",
             "-H", "Content-Type: application/json",
             "-d", job_data,
             f"{MASTER_URL}/api/v1/jobs"],
            capture_output=True,
            text=True,
            timeout=10
        )
        
        response = json.loads(result.stdout)
        job_id = response.get('id')
        
        if not job_id:
            print(f"[ERROR] No se pudo crear el job")
            print(response)
            return None
        
        print(f"[INFO] Job ID: {job_id}")
        
        # Monitorear progreso
        max_wait = 600  # 10 minutos máximo
        wait_time = 0
        status = "PENDING"
        
        while status not in ["SUCCEEDED", "FAILED"] and wait_time < max_wait:
            time.sleep(5)
            wait_time += 5
            
            try:
                result = subprocess.run(
                    ["curl", "-s", f"{MASTER_URL}/api/v1/jobs/{job_id}"],
                    capture_output=True,
                    text=True,
                    timeout=5
                )
                
                job_status = json.loads(result.stdout)
                status = job_status.get('status', 'UNKNOWN')
                progress = job_status.get('progress', 0)
                
                print(f"\r  Progreso: {progress:.1f}% ({wait_time}s transcurridos)  ", end='', flush=True)
                
            except Exception as e:
                print(f"\n[WARN] Error al obtener estado: {e}")
        
        print()  # Nueva línea
        
        end_time = time.time()
        duration = int(end_time - start_time)
        
        if status == "SUCCEEDED":
            print(f"[INFO] ✓ Job completado en {duration}s")
            
            # Calcular throughput (asumiendo tamaño conocido)
            return {
                'job_name': job_name,
                'description': description,
                'duration': duration,
                'status': status
            }
        else:
            print(f"[ERROR] ✗ Job falló o timeout (estado: {status})")
            return None
            
    except Exception as e:
        print(f"[ERROR] Error ejecutando job: {e}")
        return None

def main():
    print_header("BENCHMARK SIMPLIFICADO - MINI-SPARK")
    
    # Verificar cluster
    if not check_cluster():
        sys.exit(1)
    
    print_header("PRUEBA CON DATASET PEQUEÑO (1K registros)")
    
    # Ejecutar test de wordcount
    result = run_job(
        os.path.join(JOBS_DIR, "test_wordcount.json"),
        "WordCount Test (1K)",
        "Prueba de tokenización y conteo en 1K líneas"
    )
    
    if result:
        print_header("RESUMEN")
        print(f"Job: {result['job_name']}")
        print(f"Duración: {result['duration']}s")
        print(f"Estado: {result['status']}")
        print("\n[INFO] ✓ Prueba completada exitosamente")
        print(f"[INFO] Resultados en: {RESULTS_DIR}/")
    else:
        print("\n[ERROR] La prueba falló")
        sys.exit(1)

if __name__ == "__main__":
    main()
