# EdgeAgent-Hub

A distributed AI Agent platform for intelligent scheduling of computing and power resources.

## Features

- **AI Agent Runtime**: Secure sandboxed agent execution with multi-runtime support
- **Compute Marketplace**: Resource publishing, order management, bidding system
- **Energy Management**: Power monitoring, storage scheduling, VPP management
- **Compute-Power Coordination**: Load prediction, multi-objective optimization
- **IoT Services**: Device management, telemetry collection, protocol adaptation
- **User Management**: Authentication, authorization, billing
- **Monitoring & Alerting**: Metrics collection, alerting rules

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL or SQLite
- Docker (optional)

### Local Development

```bash
# Clone the repository
git clone https://gitcode.com/ywtech/beta.git
cd beta

# Install dependencies
go mod download

# Build
go build -o edgeagent ./cmd/edgeagent

# Run
./edgeagent --config config/config-dev.yaml
```

### Docker Deployment

```bash
docker-compose -f deploy/docker/docker-compose.yml up
```

### Kubernetes Deployment

```bash
helm install edgeagent-hub deploy/helm/edgeagent-hub
```

## API Endpoints

| Module | Base Path | Description |
|--------|-----------|-------------|
| Auth | `/api/v1/auth` | User registration, login |
| Agents | `/api/v1/agents` | Agent management |
| Assets | `/api/v1/assets` | Compute resource listing |
| Orders | `/api/v1/orders` | Order management |
| Power | `/api/v1/power` | Power source management |
| Storage | `/api/v1/storage` | Energy storage management |
| Schedule | `/api/v1/schedule` | Coordination scheduling |
| Devices | `/api/v1/devices` | IoT device management |
| Bills | `/api/v1/bills` | Billing management |
| Metrics | `/api/v1/metrics` | System metrics |

## Architecture

The platform follows a layered architecture:

- **User Layer**: Web Console, CLI, SDK, REST/gRPC API
- **Orchestration Layer**: API Gateway, Authentication, Workflow Engine
- **Capability Layer**: Agent Runtime, Marketplace, Energy, Coordination, IoT
- **Infrastructure Layer**: Kubernetes, PostgreSQL, Kafka, Redis

## License

MIT