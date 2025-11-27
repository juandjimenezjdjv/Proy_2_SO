#!/usr/bin/env python3
"""
Generador de datos de prueba para Mini-Spark
Genera CSVs con diferentes tamaños y contenidos
"""

import csv
import random
import string
from pathlib import Path

def generate_wordcount_data(filename, num_lines=1000):
    """Genera datos para wordcount"""
    words = ["spark", "hadoop", "flink", "kafka", "storm", "data", "processing", 
             "distributed", "system", "cluster", "node", "worker", "master",
             "batch", "stream", "compute", "task", "job", "dag"]
    
    with open(filename, 'w', newline='') as f:
        writer = csv.writer(f)
        writer.writerow(["id", "text", "category"])
        
        for i in range(num_lines):
            # Generar texto aleatorio con 5-15 palabras
            text = ' '.join(random.choices(words, k=random.randint(5, 15)))
            category = random.choice(["tech", "data", "cloud", "system"])
            writer.writerow([i+1, text, category])
    
    print(f"✓ Generado {filename} con {num_lines} líneas")

def generate_join_data():
    """Genera datos para probar operación join"""
    # Usuarios
    with open('../data/users.csv', 'w', newline='') as f:
        writer = csv.writer(f)
        writer.writerow(["user_id", "name", "country"])
        users = [
            (1, "Alice", "USA"),
            (2, "Bob", "UK"),
            (3, "Charlie", "Canada"),
            (4, "Diana", "USA"),
            (5, "Eve", "UK"),
        ]
        writer.writerows(users)
    
    print("✓ Generado data/users.csv")
    
    # Pedidos
    with open('../data/orders.csv', 'w', newline='') as f:
        writer = csv.writer(f)
        writer.writerow(["order_id", "user_id", "amount", "product"])
        orders = [
            (101, 1, 250, "laptop"),
            (102, 1, 50, "mouse"),
            (103, 2, 100, "keyboard"),
            (104, 3, 300, "monitor"),
            (105, 2, 75, "headset"),
            (106, 4, 500, "laptop"),
            (107, 5, 150, "tablet"),
            (108, 1, 25, "cable"),
        ]
        writer.writerows(orders)
    
    print("✓ Generado data/orders.csv")

def generate_large_dataset(filename, num_lines=10000):
    """Genera dataset grande para pruebas de performance"""
    with open(filename, 'w', newline='') as f:
        writer = csv.writer(f)
        writer.writerow(["id", "value", "timestamp", "status"])
        
        for i in range(num_lines):
            value = random.randint(1, 1000)
            timestamp = f"2025-11-{random.randint(1,30):02d}T{random.randint(0,23):02d}:{random.randint(0,59):02d}:00"
            status = random.choice(["active", "inactive", "pending", "completed"])
            writer.writerow([i+1, value, timestamp, status])
    
    print(f"✓ Generado {filename} con {num_lines} líneas")

if __name__ == "__main__":
    # Crear directorio data si no existe
    Path("../data").mkdir(exist_ok=True)
    
    print("Generando datasets de prueba...")
    
    # Dataset pequeño para pruebas rápidas
    generate_wordcount_data("../data/input.csv", num_lines=100)
    
    # Dataset mediano
    generate_wordcount_data("../data/medium-input.csv", num_lines=1000)
    
    # Dataset grande para performance
    generate_large_dataset("../data/large-input.csv", num_lines=10000)
    
    # Datos para joins
    generate_join_data()
    
    print("\n✓ Todos los datasets generados exitosamente!")
    print("\nArchivos creados:")
    print("  - data/input.csv (100 líneas)")
    print("  - data/medium-input.csv (1000 líneas)")
    print("  - data/large-input.csv (10000 líneas)")
    print("  - data/users.csv (5 usuarios)")
    print("  - data/orders.csv (8 pedidos)")
