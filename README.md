# **Moniepoint Storage Engine - Recruitment Task**

A simple persistent **Key/Value storage engine** inspired by **Bitcask**, designed for high-performance reads and writes
with optional **data replication**.

This project was developed focusing on:  
✅ **Low latency** for reads and writes  
✅ **Efficient range queries** using an **in-memory skiplist index**  
✅ **Persistence** with crash recovery  
✅ **Data replication** (leader-follower architecture)

---

## **📌 Features**

- **Key/Value store** with in-memory indexing (hash table + skiplist)
- **Efficient range queries** via the skiplist index
- **REST API** for storage operations (due to standard-library limitation)
- **Simple leader-follower replication**
- **Crash recovery** (but no data corruption checks)
- **Unit tests, stress tests, and benchmarks**

---

## **📂 Project Structure**

The project follows a **clean architecture** approach:

```
📦 moniepoint-storage-engine
├── 📂 domain              # Business logic, interfaces, entities
├── 📂 infrastructure      # External connections (filesystem, REST)
├── 📂 application         # REST & CLI apps, integration tests, benchmarks
├── 📂 cmd                 # application entrypoints
```

---

## **💾 Data Format**

Each data entry is stored as a **single line** in a file:

```
{timestamp};{key_length};{value_length};{key};{value}
```

Unlike **Bitcask**, this format **does not include checksums**, prioritizing simplicity over corruption checks.

---

## **🚀 Running the Storage Engine**

### **Standalone Mode (No Replication)**

Run the **REST API** in leader mode with a local data directory:

```sh
HTTP_PORT=8080 DATA_PATH={path-to-data} REPLICATION_MODE=leader go run cmd/rest/main.go
```

Alternatively, using **Docker**:

```sh
docker run -it --rm $(docker build -q .)
```

---

## **🔗 REST API Endpoints**

| Method   | Endpoint                               | Description                                                         |
|----------|----------------------------------------|---------------------------------------------------------------------|
| **POST** | `/put`                                 | Store a key-value pair (`{"key": "somekey", "value": "somevalue"}`) |
| **GET**  | `/read?key=somekey`                    | Retrieve a value (binary response)                                  |
| **GET**  | `/readrange?start=startkey&end=endkey` | Get a range of values (JSON response)                               |
| **POST** | `/batchput`                            | Store multiple key-value pairs (JSON object)                        |
| **POST** | `/delete?key=somekey`                  | Delete a key                                                        |

---

## **🔄 Leader-Follower Replication**

This storage engine supports **simple leader-follower replication**, where:  
✅ The **leader** writes new data and fans it out to followers  
✅ **Followers sync** with the leader at startup  
✅ Followers **forward write requests** to the leader

### **Running in Replication Mode (Docker Compose)**

Start a **leader** and **4 followers** using:

```sh
docker compose up
```

The **leader** is available at `http://localhost:8080`, while followers are at ports `8081-8084`.

---

## **🛠️ Testing & Benchmarking**

### **Unit Tests**

Run all **unit tests** (excluding stress tests):

```sh
go test --short ./...
```

### **Stress Tests**

1️⃣ Start the REST API  
2️⃣ Run stress tests:

```sh
go test application/integration_tests/rest_stress_test.go
```

### **Benchmarks**

```sh
go test -bench=. ./application/integration_tests/bench_storage_engine_test.go
```

---

## **🔍 Limitations & Future Improvements**

✅ **No voting mechanism** (no automatic leader election)  
✅ **No checksum verification** (risk of corruption)  
✅ **Basic test coverage** (only happy paths covered)  
✅ **REST instead of gRPC** (due to no external libraries requirement)  
✅ **Potential future RAFT-based leader election**  
✅ **Potential migration from in-memory indexes to LSM-Tree**

---

## **📜 Summary**

This project delivers a **simple yet efficient** persistent key-value store with optional **leader-follower replication
**.  
It trades off **data integrity checks** for **ease of implementation**, but achieves **high-performance storage** and *
*efficient range queries**.
---
