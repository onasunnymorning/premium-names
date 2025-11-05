# Domain Labels Management System

A comprehensive system for processing, managing, and analyzing domain labels with advanced tagging and Temporal workflow processing capabilities.

## Features

### Backend (Go + Temporal + PostgreSQL)
- **Temporal Workflows**: Scalable, fault-tolerant domain label processing
- **Multiple File Formats**: Support for CSV, TSV, and Excel files
- **Smart Label Extraction**: Automatically extracts domain labels from various input formats (URLs, FQDNs, labels)
- **Tagging System**: Flexible tagging with statistics and filtering
- **RESTful API**: Complete CRUD operations for labels and tags
- **Database Integration**: PostgreSQL with GORM for robust data management

### Frontend (Next.js + TypeScript + TailwindCSS)
- **Modern Dashboard**: Clean interface for viewing and managing domain labels
- **Advanced Search & Filtering**: Search labels by text, filter by tags, creator, etc.
- **File Upload**: Drag-and-drop interface with real-time processing feedback
- **Responsive Design**: Works on desktop, tablet, and mobile devices
- **Real-time Updates**: React Query for efficient data fetching and caching

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Next.js       │    │   Go API        │    │   Temporal      │
│   Frontend      │───▶│   (Gin/GORM)    │───▶│   Worker        │
│   (Port 3000)   │    │   (Port 8081)   │    │   (Port 7233)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │   PostgreSQL    │    │   File Storage  │
                       │   Database      │    │   (S3/Local)    │
                       │   (Port 5433)   │    │                 │
                       └─────────────────┘    └─────────────────┘
```

## Getting Started

### Prerequisites
- Docker and Docker Compose
- Go 1.21+ (for local development)
- Node.js 18+ (for frontend development)

### Quick Start with Docker

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd premium-names
   ```

2. **Start the complete stack**
   ```bash
   make dev-up
   ```

   This starts:
   - **Temporal Server** (UI at http://localhost:8080)
   - **PostgreSQL Database** (Port 5433)
   - **MinIO S3** (Console at http://localhost:9001)
   - **Go API Server** (Port 8081)
   - **Next.js Frontend** (Port 3000)
   - **Temporal Worker**

3. **Access the application**
   - **Frontend**: http://localhost:3000
   - **API**: http://localhost:8081
   - **Temporal UI**: http://localhost:8080
   - **MinIO Console**: http://localhost:9001 (admin/minioadmin)

### Manual Development Setup

#### Backend Development
```bash
# Install dependencies
go mod tidy

# Set environment variables
export DB_HOST=localhost
export DB_PORT=5433
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=domain_labels
export TEMPORAL_ADDRESS=localhost:7233

# Start the API server
go run cmd/api/main.go

# Start the Temporal worker (in separate terminal)
go run cmd/worker/main.go
```

#### Frontend Development
```bash
cd frontend

# Install dependencies
npm install

# Set environment variables
echo "NEXT_PUBLIC_API_URL=http://localhost:8081/api/v1" > .env.local

# Start development server
npm run dev
```

## Usage

### 1. Upload Domain Labels

1. Navigate to the **Upload** tab
2. Choose processing method:
   - **Direct Upload**: Immediate processing (good for smaller files)
   - **Workflow Processing**: Temporal-based processing (scalable for large files)
3. Drag and drop or select your file (CSV, TSV, Excel)
4. Add tags (comma-separated)
5. Enter your email
6. Click **Upload**

#### File Format Requirements
- **First column** must contain domain names, labels, or URLs
- Supports headers (automatically detected and skipped)
- Handles empty lines gracefully
- Accepts various formats:
  - Domain labels: `example`, `test-site`
  - Full domains: `example.com`, `test-site.org`
  - URLs: `https://example.com`, `http://test.site.net`

#### Example CSV File
```csv
Domain,Description,Category
example,Test domain,premium
test-site,Development site,standard
https://my-app.com,Production app,premium
sub.domain.org,Subdomain example,standard
```

### 2. Browse and Search Labels

1. Navigate to the **Dashboard** tab
2. Use the search bar to find specific labels
3. Click on tag filters to filter by specific tags
4. Use pagination to browse through large datasets
5. Edit or delete labels as needed

#### Search Features
- **Text Search**: Search in label names
- **Tag Filtering**: Filter by one or more tags
- **Creator Filtering**: Filter by who created the labels
- **Sorting**: Sort by creation date, label name, etc.
- **Pagination**: Navigate through large result sets

### 3. Temporal Workflows

The system uses Temporal for robust, scalable file processing:

#### Zone Names Workflow (Existing)
Processes DNS zone files to extract unique domain names.

#### Domain Labels Workflow (New)
Processes uploaded files to extract and save domain labels:

1. **Parse File**: Reads CSV/TSV/Excel and extracts labels from first column
2. **Normalize Labels**: Converts URLs/domains to normalized labels
3. **Save to Database**: Creates labels and applies tags
4. **Handle Duplicates**: Merges tags for existing labels

#### Monitoring Workflows
- **Temporal UI**: http://localhost:8080 - View workflow executions, status, and history
- **Metrics**: http://localhost:9090/metrics - Prometheus metrics for monitoring
- **API Status**: Check workflow status via `/api/v1/workflows/{id}/status`

## API Reference

### Labels API

#### Get Labels (with pagination and filtering)
```http
GET /api/v1/labels?page=1&limit=25&q=example&tag=premium&created_by=user@example.com
```

#### Get Single Label
```http
GET /api/v1/labels/{id}
```

#### Update Label Tags
```http
PUT /api/v1/labels/{id}/tags
Content-Type: application/json

{
  "tags": ["premium", "updated"]
}
```

#### Delete Label
```http
DELETE /api/v1/labels/{id}
```

### Tags API

#### Get All Tags
```http
GET /api/v1/tags
```

#### Get Tag Statistics
```http
GET /api/v1/tags/stats
```

### Workflows API

#### Start Domain Labels Workflow
```http
POST /api/v1/workflows/domain-labels
Content-Type: application/json

{
  "file_uri": "s3://bucket/file.csv",
  "tags": ["import-2024", "premium"],
  "created_by": "user@example.com",
  "description": "Monthly premium domains import"
}
```

#### Get Workflow Status
```http
GET /api/v1/workflows/{workflow-id}/status
```

### Traditional Upload API

#### Upload File (Direct Processing)
```http
POST /api/v1/upload
Content-Type: multipart/form-data

file: <csv/tsv/excel file>
tags: premium,import-2024
created_by: user@example.com
```

## Configuration

### Environment Variables

#### Backend
- `DB_HOST`: Database host (default: localhost)
- `DB_PORT`: Database port (default: 5432)
- `DB_USER`: Database username (default: postgres)
- `DB_PASSWORD`: Database password (default: postgres)
- `DB_NAME`: Database name (default: domain_labels)
- `TEMPORAL_ADDRESS`: Temporal server address (default: localhost:7233)
- `PORT`: API server port (default: 8080)

#### Frontend
- `NEXT_PUBLIC_API_URL`: Backend API URL (default: http://localhost:8081/api/v1)

#### Temporal Worker
- `TEMPORAL_TARGET_HOST`: Temporal server host
- `TEMPORAL_NAMESPACE`: Temporal namespace (default: default)
- `TEMPORAL_TASK_QUEUE`: Task queue name (default: zone-names)
- `ZN_TMP_DIR`: Temporary directory for file processing

## Development

### Project Structure
```
premium-names/
├── cmd/
│   ├── api/           # API server main
│   └── worker/        # Temporal worker main
├── internal/
│   ├── activities/    # Temporal activities
│   ├── api/          # HTTP handlers
│   ├── db/           # Database connection
│   ├── domain/       # Domain logic
│   ├── models/       # Database models
│   ├── types/        # Type definitions
│   └── workflow/     # Temporal workflows
├── frontend/         # Next.js frontend application
├── web/             # Static web assets (legacy)
└── docker-compose.yml
```

### Running Tests
```bash
# Backend tests
go test ./...

# Frontend tests  
cd frontend
npm test
```

### Building for Production
```bash
# Backend
make build

# Frontend
cd frontend
npm run build
```

## Troubleshooting

### Common Issues

1. **Database Connection Issues**
   - Check PostgreSQL is running: `docker-compose ps`
   - Verify connection settings in environment variables
   - Check logs: `docker-compose logs db`

2. **Temporal Connection Issues**
   - Ensure Temporal server is running: `docker-compose logs temporal`
   - Check worker logs: `docker-compose logs worker`
   - Verify TEMPORAL_ADDRESS environment variable

3. **File Upload Issues**
   - Check file format (CSV, TSV, Excel only)
   - Verify file size limits (default: 8MB)
   - Check API logs for parsing errors

4. **Frontend API Issues**
   - Verify NEXT_PUBLIC_API_URL is correct
   - Check CORS settings in backend
   - Check browser network tab for API errors

### Logs and Debugging
```bash
# View all services
docker-compose logs -f

# View specific service
docker-compose logs -f api
docker-compose logs -f frontend  
docker-compose logs -f worker

# View Temporal workflows
# Go to http://localhost:8080
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details.
