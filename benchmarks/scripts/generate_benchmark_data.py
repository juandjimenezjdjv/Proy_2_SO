#!/usr/bin/env python3
"""
Script para generar dataset de benchmark de 1M registros CSV
"""

import csv
import random
import sys
from datetime import datetime, timedelta

def generate_large_dataset(filename, num_records=1000000, num_partitions=1):
    """
    Genera un dataset grande para benchmarks
    
    Args:
        filename: Nombre base del archivo (sin extensión)
        num_records: Número total de registros a generar
        num_partitions: Número de particiones a crear
    """
    
    # Listas para generar datos realistas
    products = [
        "laptop", "smartphone", "tablet", "monitor", "keyboard", 
        "mouse", "headphones", "webcam", "router", "printer",
        "scanner", "speaker", "microphone", "charger", "cable"
    ]
    
    categories = ["electronics", "accessories", "peripherals", "networking", "office"]
    
    cities = [
        "New York", "Los Angeles", "Chicago", "Houston", "Phoenix",
        "Philadelphia", "San Antonio", "San Diego", "Dallas", "San Jose"
    ]
    
    statuses = ["pending", "shipped", "delivered", "cancelled"]
    
    records_per_partition = num_records // num_partitions
    
    print(f"Generando {num_records:,} registros en {num_partitions} particiones...")
    
    for partition in range(num_partitions):
        if num_partitions == 1:
            output_file = f"{filename}.csv"
        else:
            output_file = f"{filename}-part-{partition}.csv"
        
        start_idx = partition * records_per_partition
        end_idx = start_idx + records_per_partition if partition < num_partitions - 1 else num_records
        
        print(f"  Creando {output_file} ({end_idx - start_idx:,} registros)...")
        
        with open(output_file, 'w', newline='') as f:
            writer = csv.writer(f)
            
            # Header
            writer.writerow([
                'id', 'timestamp', 'product', 'category', 
                'quantity', 'price', 'city', 'status'
            ])
            
            # Data
            base_date = datetime(2024, 1, 1)
            
            for i in range(start_idx, end_idx):
                record_id = f"ORD-{i:08d}"
                timestamp = (base_date + timedelta(seconds=i)).strftime("%Y-%m-%d %H:%M:%S")
                product = random.choice(products)
                category = random.choice(categories)
                quantity = random.randint(1, 10)
                price = round(random.uniform(10.0, 1000.0), 2)
                city = random.choice(cities)
                status = random.choice(statuses)
                
                writer.writerow([
                    record_id, timestamp, product, category,
                    quantity, price, city, status
                ])
        
        print(f"  ✓ {output_file} creado exitosamente")
    
    print(f"\n✅ Dataset completo generado: {num_records:,} registros")

def generate_wordcount_dataset(filename, num_records=1000000):
    """
    Genera un dataset para benchmark de wordcount
    
    Args:
        filename: Nombre del archivo de salida
        num_records: Número de líneas de texto
    """
    
    words = [
        "spark", "hadoop", "kafka", "flink", "storm",
        "processing", "distributed", "data", "stream", "batch",
        "map", "reduce", "filter", "transform", "aggregate",
        "cluster", "node", "master", "worker", "task"
    ]
    
    print(f"Generando dataset de wordcount con {num_records:,} líneas...")
    
    with open(filename, 'w') as f:
        f.write("text\n")  # Header
        
        for i in range(num_records):
            # Cada línea tiene entre 5 y 15 palabras
            line_words = random.choices(words, k=random.randint(5, 15))
            f.write(" ".join(line_words) + "\n")
            
            if (i + 1) % 100000 == 0:
                print(f"  Generadas {i + 1:,} líneas...")
    
    print(f"✅ Dataset de wordcount generado: {num_records:,} líneas")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Uso:")
        print("  python generate_benchmark_data.py <tipo> [opciones]")
        print("\nTipos:")
        print("  sales <archivo> <registros> <particiones>")
        print("    Ejemplo: python generate_benchmark_data.py sales data/sales 1000000 4")
        print("\n  wordcount <archivo> <lineas>")
        print("    Ejemplo: python generate_benchmark_data.py wordcount data/wordcount.csv 1000000")
        sys.exit(1)
    
    dataset_type = sys.argv[1]
    
    if dataset_type == "sales":
        filename = sys.argv[2] if len(sys.argv) > 2 else "benchmark_data"
        num_records = int(sys.argv[3]) if len(sys.argv) > 3 else 1000000
        num_partitions = int(sys.argv[4]) if len(sys.argv) > 4 else 1
        
        generate_large_dataset(filename, num_records, num_partitions)
        
    elif dataset_type == "wordcount":
        filename = sys.argv[2] if len(sys.argv) > 2 else "wordcount_data.csv"
        num_records = int(sys.argv[3]) if len(sys.argv) > 3 else 1000000
        
        generate_wordcount_dataset(filename, num_records)
        
    else:
        print(f"❌ Tipo de dataset desconocido: {dataset_type}")
        print("Tipos válidos: sales, wordcount")
        sys.exit(1)
