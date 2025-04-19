# Atomic Counter Demo

A distributed, fault-tolerant counter service built with Go, utilizing the Raft consensus algorithm for replication and Gossip-based cluster discovery.

## Features

- **Distributed Counter Service**: Maintains a consistent counter value across multiple nodes.
- **Raft Consensus**: Ensures strong consistency and fault tolerance.
- **Gossip Protocol**: Facilitates dynamic cluster membership and node discovery.
- **HTTP API**: Provides endpoints for incrementing and retrieving the counter value.

## Architecture

The system comprises multiple nodes that communicate over HTTP and Gossip protocols. Each node participates in the Raft consensus to replicate the counter state, ensuring consistency even in the presence of node failures.

## Getting Started

### Prerequisites

- Go 1.16 or higher
- Unix-like operating system (Linux, macOS)

### Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/arashnikoo/atomic-counter-demo.git
   cd atomic-counter-demo
   ```

2. Build the project:

   ```bash
   go build -o atomic-counter
   ```

### Running the Service

You can start multiple nodes to form a cluster. Use the provided `start.sh` script to launch nodes with appropriate configurations.

```bash
# starting node 1
./start.sh 1 

# starting node 2
./start.sh 2

# starting node 3
./start.sh 3
```

This script initializes multiple nodes, each listening on different ports and configured to discover peers via the Gossip protocol.

### API Usage

Once the nodes are running, you can interact with the counter service using HTTP requests.

- **Increment Counter**:

  ```bash
  curl -X POST http://localhost:9000/next
  ```

- **Get Counter Value**:

  ```bash
  curl http://localhost:9000/get
  ```

Replace `9000` with the port number of the node you wish to interact with.

## Project Structure

- `main.go`: Entry point of the application.
- `config/`: Configuration management.
- `gossip/`: Implementation of the Gossip protocol for node discovery.
- `http/`: HTTP server and API handlers.
- `raft/`: Raft consensus algorithm implementation.
- `types/`: Shared data structures.
- `utils/`: Utility functions and data types
- `start.sh`: Script to start multiple nodes for testing.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
