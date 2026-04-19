# 🚀 DevCost AI
**AI-Powered Cloud Cost Optimization Platform for AWS, GCP, and Azure**

## 📌 Overview

DevCost AI is an intelligent cloud cost optimization platform that automatically detects wasteful resource usage, generates actionable recommendations, and provides AI-powered insights to reduce cloud spending by up to 30%. Built for DevOps engineers, SREs, and engineering teams who need data-driven cost control without manual analysis overhead.

The platform continuously monitors your cloud infrastructure, identifies cost inefficiencies using advanced algorithms, and provides both automated actions and human-readable explanations powered by local AI models.

## 🌍 Problem Statement

**Cloud costs are spiraling out of control**, and most organizations lack visibility into where money is being wasted:

- **Silent Resource Waste**: Idle EC2 instances, oversized RDS databases, and unused storage volumes drain budgets continuously
- **Manual Cost Analysis**: Engineers spend hours analyzing CloudWatch metrics, usage patterns, and billing data instead of building features  
- **Reactive Cost Management**: Teams discover cost spikes after they've already impacted budgets, often weeks or months later
- **Complex Optimization**: Determining safe cost optimizations requires deep AWS knowledge and risk assessment that teams don't have time for
- **No Actionable Insights**: Existing tools show charts but don't explain *why* costs increased or *what* specific actions to take

**The result**: Engineering teams overspend by 25-40% on cloud infrastructure while lacking the tools and time to optimize effectively.

## 💡 Solution

DevCost AI solves cloud cost optimization through **automated detection, intelligent recommendations, and AI-powered explanations**:

### **Core Innovation**
- **Waste Detection Engine**: Continuously monitors AWS resources using configurable thresholds (CPU < 5% = idle, oversized databases, unused volumes)
- **Intelligent Recommendations**: Generates specific, risk-assessed actions (right-size, schedule shutdowns, migrate storage classes)  
- **AI-Powered Explanations**: Local Ollama integration provides human-readable analysis: *"This EC2 instance costs $240/month but averages 2% CPU usage. Stopping it saves $240/month with zero business impact."*
- **Automated Execution**: One-click implementation of safe optimizations with audit trails and rollback capability
- **Team Integration**: Slack notifications and dashboard alerts keep teams informed without overwhelming them

### **Key Differentiator**
Unlike generic monitoring tools, DevCost AI focuses specifically on **cost optimization with safety guarantees**—every recommendation includes risk assessment, estimated savings, and clear explanations that non-experts can understand and act upon.

## ✨ Features

- 🔍 **Multi-Cloud Waste Detection** - Automated scanning for idle instances, oversized resources, and unused storage
- 📊 **Real-Time Cost Dashboard** - Executive-level metrics: total cost, waste percentage, monthly savings potential
- 🤖 **AI-Powered Analysis** - Natural language explanations of cost spikes and optimization opportunities via Ollama
- ⚡ **One-Click Optimizations** - Execute safe cost reductions with confidence scoring and rollback capability
- 📈 **Historical Tracking** - Track savings over time with detailed action logs and impact analysis
- 🔔 **Slack Integration** - `/cost explain` and `/cost why` commands for team cost awareness
- 🛡️ **Safety First** - Risk assessment prevents business-critical resource modifications

## 🏗️ Architecture

DevCost AI uses a modern microservices architecture designed for scalability and reliability:

```
                              ┌─────────────────┐
                              │   Users/Teams   │
                              └─────────┬───────┘
                                        │
                  ┌─────────────────────┼─────────────────────┐
                  │                     │                     │
          ┌───────▼────────┐    ┌───────▼────────┐    ┌──────▼─────────┐
          │  Next.js Web   │    │  Slack Bot     │    │  REST API      │
          │  Dashboard     │    │  Integration   │    │  Clients       │
          │  (Port 3000)   │    │  (/cost cmd)   │    │  (Port 8080)   │
          └────────────────┘    └────────────────┘    └────────────────┘
                  │                     │                     │
                  └─────────────────────┼─────────────────────┘
                                        │
                              ┌─────────▼───────────┐
                              │   Go Backend API    │
                              │   (Gin Framework)   │
                              │                     │
                              │ ┌─────────────────┐ │
                              │ │ Waste Detection │ │ 
                              │ │    Service      │ │
                              │ └─────────────────┘ │
                              │ ┌─────────────────┐ │
                              │ │ Recommendation  │ │
                              │ │    Service      │ │
                              │ └─────────────────┘ │
                              │ ┌─────────────────┐ │
                              │ │ Action Pipeline │ │
                              │ │    Service      │ │
                              │ └─────────────────┘ │
                              └─────────┬───────────┘
                                        │
                  ┌─────────────────────┼─────────────────────┐
                  │                     │                     │
          ┌───────▼────────┐    ┌───────▼────────┐    ┌──────▼─────────┐
          │  PostgreSQL    │    │     Redis      │    │  Ollama AI     │
          │  (Metrics &    │    │   (Caching)    │    │ (Local LLM)    │
          │   Actions)     │    └────────────────┘    └────────────────┘
          └────────────────┘            │
                  │                     │
          ┌───────▼───────────────┐     │
          │                       │     │
          │    External APIs      │     │
          │                       │     │
          │  ┌─────────────────┐  │     │
          │  │   AWS APIs      │  │◄────┘
          │  │ (EC2, RDS, S3)  │  │
          │  └─────────────────┘  │
          │  ┌─────────────────┐  │
          │  │   GCP APIs      │  │
          │  │ (Future)        │  │
          │  └─────────────────┘  │
          │  ┌─────────────────┐  │
          │  │  Azure APIs     │  │
          │  │ (Future)        │  │
          │  └─────────────────┘  │
          └───────────────────────┘
```

### **Component Breakdown**

| Component | Purpose | Technology | Port |
|-----------|---------|------------|------|
| **Frontend** | Cost dashboard, waste visualization, team interface | Next.js 14 + Tailwind CSS | 3000 |
| **Backend API** | Business logic, AWS integration, data processing | Go 1.21 + Gin Framework | 8080 |
| **PostgreSQL** | Resource metadata, recommendations, action logs | PostgreSQL 15 | 5432 |
| **Redis** | API response caching, session storage | Redis 7 | 6379 |
| **Ollama AI** | Local LLM for cost analysis explanations | Ollama (llama3.2) | 11434 |

## 🔄 End-to-End Flow

Here's how a typical cost optimization cycle works through the system:

### **1. Resource Discovery** 
```
AWS APIs → Backend → PostgreSQL
```
- **Scheduler** runs every 6 hours to scan AWS resources
- **Resource Service** fetches EC2 instances, RDS databases, S3 buckets via AWS SDK
- **Database** stores resource metadata: `type`, `region`, `size`, `cost_per_hour`

### **2. Waste Detection**
```
PostgreSQL → Waste Detection Service → PostgreSQL
```
- **WasteDetectionService** applies intelligent thresholds:
  - **Idle EC2**: CPU < 5% for 24+ hours
  - **Oversized RDS**: CPU < 20% + storage utilization < 50%  
  - **Unused EBS**: Unattached volumes > 30 days
- **Results**: Stored as `waste_results` with confidence scores and estimated savings

### **3. Recommendation Generation**  
```
Waste Results → Recommendation Service → PostgreSQL
```
- **RecommendationService** creates actionable suggestions:
  - *Stop idle instances during business hours (saves $240/month)*
  - *Downsize RDS from db.t3.large to db.t3.medium (saves $80/month)*
- **Risk Assessment**: Each recommendation tagged with risk level (low/medium/high)

### **4. AI Analysis** (Optional)
```
Recommendation → Ollama AI → Enhanced Description
```
- **AI Service** generates human-readable explanations
- **Example**: *"EC2 instance i-abc123 runs constantly but only uses 2% CPU. This pattern indicates it's either misconfigured or the workload could run on a smaller instance, saving $240 monthly."*

### **5. User Interaction**
```
Frontend Dashboard → User Decision → Action Execution
```
- **Dashboard** displays total savings potential, waste breakdown by service
- **User** reviews recommendations and selects actions to execute
- **Action Pipeline** executes changes via AWS APIs with audit logging

### **6. Impact Tracking**
```
Executed Actions → Database → Dashboard Updates
```
- **Post-execution monitoring** validates savings were achieved
- **Historical tracking** shows cost reduction trends over time
- **Team notifications** via Slack confirm successful optimizations

## 🛠️ Tech Stack

### **Frontend**
- **Next.js 14** - React framework with server-side rendering
- **TypeScript** - Type-safe development
- **Tailwind CSS** - Utility-first styling
- **React Hook Form** - Form handling and validation

### **Backend**
- **Go 1.21** - High-performance API development
- **Gin Framework** - HTTP web framework
- **GORM** - Object-relational mapping
- **go-playground/validator** - Request validation
- **golang-migrate/migrate** - Database migrations

### **Data & Storage**
- **PostgreSQL 15** - Primary data store for metrics and configurations
- **Redis 7** - Caching layer for API responses and sessions

### **DevOps & Infrastructure**
- **Docker** - Containerization for all services
- **Docker Compose** - Local development orchestration
- **GitHub Actions** - CI/CD pipelines (planned)
- **Nginx** - Reverse proxy (production deployment)

### **Cloud Integrations**
- **AWS SDK for Go** - EC2, RDS, S3, CloudWatch APIs
- **Ollama** - Local LLM for AI-powered explanations
- **Slack SDK** - Team notifications and bot commands

### **Monitoring & Observability**
- **Structured logging** - JSON-formatted application logs
- **Health checks** - Service availability monitoring
- **Graceful shutdown** - Clean container termination

## ⚙️ Setup & Installation

### **Prerequisites**
- Docker & Docker Compose
- Git
- 8GB RAM (for local AI model)

### **Quick Start**

1. **Clone the repository**
   ```bash
   git clone https://github.com/yourusername/devcost-ai.git
   cd devcost-ai
   ```

2. **Setup environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration (see Environment Variables section)
   ```

3. **Start the application**
   ```bash
   docker-compose up -d
   ```

4. **Access the services**
   - **Dashboard**: http://localhost:3000
   - **API**: http://localhost:8080/health
   - **Database Admin**: http://localhost:5050 (optional)

### **Development Setup**

For local development without Docker:

1. **Start PostgreSQL and Redis**
   ```bash
   docker-compose up -d postgres redis
   ```

2. **Run backend**
   ```bash
   cd backend
   go mod download
   go run main.go
   ```

3. **Run frontend** 
   ```bash
   cd frontend
   npm install
   npm run dev
   ```



### Required Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `postgres` | Database hostname |
| `DB_PORT` | `5432` | Database port |
| `DB_NAME` | `devcost_ai` | Database name |
| `DB_USER` | `devcost` | Database user |
| `DB_PASSWORD` | `devcost123` | Database password |
| `LOG_LEVEL` | `info` | Logging level (debug/info/warn/error) |
| `AWS_ACCESS_KEY_ID` | - | AWS access key (required for AWS integration) |
| `AWS_SECRET_ACCESS_KEY` | - | AWS secret key (required for AWS integration) |
| `AWS_REGION` | `us-east-1` | Default AWS region |
| `AI_ENABLED` | `false` | Enable AI-powered analysis (requires Ollama) |
| `AI_BASE_URL` | `http://localhost:11434` | Ollama server URL |
| `SLACK_BOT_TOKEN` | - | Slack bot token for integrations |

## 📊 Features

- **Dashboard**: Overview of costs, waste, and savings opportunities
- **Waste Detection**: Automatically identify:
  - Idle EC2 instances (CPU < 5% for 24+ hours)
  - Unattached EBS volumes

## 🔐 Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| **Application** | | | |
| `APP_ENV` | Environment mode | `development` | ✅ |
| `LOG_LEVEL` | Logging verbosity | `info` | ✅ |
| **Database** | | | |
| `DB_HOST` | PostgreSQL hostname | `postgres` | ✅ |
| `DB_PORT` | PostgreSQL port | `5432` | ✅ |
| `DB_NAME` | Database name | `devcost_ai` | ✅ |
| `DB_USER` | Database username | `devcost` | ✅ |
| `DB_PASSWORD` | Database password | `devcost123` | ✅ |
| **AWS** (Required for cloud integration) | | | |
| `AWS_ACCESS_KEY_ID` | AWS access key | - | ⚠️ |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key | - | ⚠️ |
| `AWS_REGION` | AWS region | `us-east-1` | ⚠️ |
| **AI (Optional)** | | | |
| `AI_ENABLED` | Enable AI explanations | `false` | ❌ |
| `AI_BASE_URL` | Ollama server URL | `http://localhost:11434` | ❌ |
| `AI_MODEL` | AI model name | `llama3.2` | ❌ |
| **Slack (Optional)** | | | |
| `SLACK_BOT_TOKEN` | Bot token for API access | - | ❌ |
| `SLACK_SIGNING_SECRET` | Request verification | - | ❌ |
| `SLACK_WEBHOOK_URL` | Notification webhook | - | ❌ |

**Legend**: ✅ Required, ⚠️ Required in production, ❌ Optional

## 📦 Running the Project

### **Quick Start**
```bash
# Copy environment file and configure
cp .env.example .env
# Edit .env with your AWS credentials

# Start all services
docker-compose up -d

# Monitor startup
docker-compose logs -f backend
```

### **Development Mode**
```bash
# Start dependencies only
docker-compose up -d postgres redis

# Run backend locally
cd backend && go run main.go

# Run frontend locally  
cd frontend && npm run dev
```

### **AI-Enhanced Mode**
```bash
# Install and start Ollama
curl -fsSL https://ollama.ai/install.sh | sh
ollama pull llama3.2

# Enable AI in .env
AI_ENABLED=true

# Restart backend
docker-compose restart backend
```

### **Expected Behavior**

After successful startup, you should see:

1. **Dashboard** (http://localhost:3000):
   - Cost overview cards with metrics
   - Navigation to Waste, Recommendations, Actions pages

2. **API Health** (http://localhost:8080/health):
   ```json
   {
     "status": "healthy",
     "timestamp": "2026-04-01T05:15:00Z",
     "database": "connected",
     "redis": "connected"
   }
   ```

3. **Database**: Auto-migrated schema with initial tables
4. **Logs**: Structured JSON output showing service initialization

## 🧪 Testing

### **API Testing**
```bash
# Health check
curl http://localhost:8080/health

# List resources
curl http://localhost:8080/api/v1/resources

# Get waste analysis
curl http://localhost:8080/api/v1/waste
```

### **Database Testing**
```bash
# Connect to database
docker exec -it devcost-postgres psql -U devcost -d devcost_ai

# Check tables
\dt

# Sample data queries
SELECT COUNT(*) FROM resources;
SELECT * FROM waste_results LIMIT 5;
```

## 🚧 Challenges & Design Decisions

### **Key Technical Trade-offs**

**1. Local AI vs Cloud AI Services**
- **Decision**: Ollama for local LLM instead of OpenAI/Anthropic APIs
- **Reasoning**: Cost control (no per-request fees), data privacy, offline capability
- **Trade-off**: Requires more resources (8GB RAM) but eliminates ongoing AI costs

**2. Golang Backend vs Node.js**
- **Decision**: Go for backend despite JavaScript frontend
- **Reasoning**: Superior performance for AWS API calls, better resource utilization, stronger typing
- **Trade-off**: Multi-language codebase but optimized for concurrent cloud API processing

**3. PostgreSQL vs Time-Series Database**
- **Decision**: PostgreSQL for all data instead of InfluxDB/TimescaleDB
- **Reasoning**: Simpler operations, JSONB support for flexible schemas, familiar to most teams
- **Trade-off**: Less optimized for time-series queries but easier deployment and maintenance

### **Safety-First Approach**

**Risk Assessment Framework**: Every recommendation includes risk scoring to prevent business disruption:
- **Low Risk**: Stop clearly idle instances, remove unattached volumes
- **Medium Risk**: Resize instances with consistent low usage patterns  
- **High Risk**: Database modifications, production instance changes

**Rollback Capability**: All actions store original configurations for one-click reversal.

## 🔮 Future Improvements

### **Near-term Enhancements (Next 6 months)**
- **Multi-cloud Support**: GCP and Azure integration using similar detection algorithms  
- **Advanced Scheduling**: Intelligent instance scheduling based on usage patterns (auto-stop evenings/weekends)
- **Cost Forecasting**: ML-powered predictions of future spend based on current trends
- **Team Budgets**: Department-level cost allocation and budget alerts
- **Enhanced UI**: Cost trend graphs, savings tracking over time, advanced filtering

### **Medium-term Goals (6-12 months)**  
- **Kubernetes Integration**: Pod rightsizing recommendations and resource limit optimization
- **Custom Policies**: User-defined waste detection rules and thresholds per environment
- **Approval Workflows**: Multi-stage approval for high-risk optimizations
- **Cost Attribution**: Automatic tagging and cost allocation by team/project/environment  
- **Mobile Dashboard**: React Native app for executive cost oversight

### **Long-term Vision (12+ months)**
- **Predictive Optimization**: Proactive scaling recommendations before usage spikes
- **FinOps Integration**: Integration with CloudHealth, Cloudability, and other FinOps tools
- **Terraform Integration**: Infrastructure-as-code cost analysis and optimization suggestions
- **Advanced AI**: Multi-model AI analysis for complex optimization scenarios

## 🤝 Contributing

We welcome contributions! DevCost AI is designed to be community-driven.

### **Development Process**
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes following our coding standards
4. Add tests for new functionality
5. Ensure all tests pass (`go test ./...` and `npm test`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)  
8. Open a Pull Request

### **Code Standards**
- **Go**: Follow `gofmt`, use meaningful variable names, add comments for complex logic
- **TypeScript**: Use strict mode, define proper interfaces, prefer functional components  
- **Database**: Use migrations for schema changes, never modify existing migrations
- **Documentation**: Update README and code comments for new features

### **Areas for Contribution**
- 🔌 **Cloud Integrations**: GCP, Azure, DigitalOcean support
- 🤖 **AI Enhancements**: Better cost analysis prompts and models
- 📊 **Visualization**: More detailed cost breakdown charts and graphs
- 🧪 **Testing**: Increase code coverage, add integration tests
- 📖 **Documentation**: API documentation, deployment guides

See `CONTRIBUTING.md` for detailed guidelines.

## 📜 License

MIT License - see `LICENSE` file for details.

---

**DevCost AI** - Making cloud cost optimization intelligent, automated, and accessible.

*Built with ❤️ for engineering teams who want to focus on building, not cost analysis.*
