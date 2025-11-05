# Domain Labels Workflow Examples

This directory contains example JSON files for starting the `DomainLabelsWorkflow` via Temporal CLI (`tctl`).

## 📁 Example Files

### 1. `domain-labels.example.json` - **S3 Premium Domains**
```json
{
  "file_uri": "s3://domain-files/premium-domains-2024-11.csv",
  "tags": ["premium", "november-2024", "curated"],
  "created_by": "admin@centralnic.com",
  "description": "Monthly premium domain names import for November 2024"
}
```
**Use case**: Processing premium domain lists stored in S3/MinIO

### 2. `domain-labels-local.example.json` - **Local File Analysis** 
```json
{
  "file_uri": "file:///Users/admin/domains/competitor-analysis.xlsx",
  "tags": ["competitor", "analysis", "q4-2024"],
  "created_by": "research@centralnic.com",
  "description": "Competitor domain portfolio analysis for Q4 2024"
}
```
**Use case**: Processing local Excel/CSV files for analysis

### 3. `domain-labels-bulk.example.json` - **TLD Zone Processing**
```json
{
  "file_uri": "s3://bulk-imports/tld-analysis/net-zone-sample.tsv",
  "tags": ["tld-analysis", "net", "zone-file", "bulk-import"],
  "created_by": "system@centralnic.com", 
  "description": "Automated .NET TLD zone file analysis - extracting premium candidates"
}
```
**Use case**: Bulk processing of TLD zone files

## 🚀 How to Use

### Method 1: Using Make Commands

```bash
# Use default example (S3 premium domains)
make start-domain-labels-workflow

# Use specific example
make start-domain-labels-workflow DOMAIN_INPUT=examples/domain-labels-local.example.json

# Use bulk processing example  
make start-domain-labels-workflow DOMAIN_INPUT=examples/domain-labels-bulk.example.json
```

### Method 2: Direct tctl Commands

```bash
# Set Temporal connection (if not using defaults)
export TEMPORAL_ADDRESS=localhost:7233
export TEMPORAL_NAMESPACE=default

# Start workflow with specific example
tctl workflow start \
  --taskqueue zone-names \
  --workflow_type DomainLabelsWorkflow \
  --input_file examples/domain-labels.example.json
```

### Method 3: Via API (Programmatic)

```bash
# Start via REST API
curl -X POST http://localhost:8081/api/v1/workflows/domain-labels \
  -H "Content-Type: application/json" \
  -d @examples/domain-labels.example.json
```

## 📊 Monitoring Workflow Execution

### 1. **Temporal UI** (Recommended)
```bash
# Open in browser
open http://localhost:8080

# Navigate to:
# - Workflows → Filter by "DomainLabelsWorkflow"  
# - Click on specific workflow run for details
# - View activities, retries, and progress
```

### 2. **CLI Monitoring**
```bash
# List all workflows
tctl workflow list

# Show workflow details  
tctl workflow show --workflow_id <workflow-id>

# Follow workflow progress
tctl workflow observe --workflow_id <workflow-id>
```

### 3. **API Status Check**
```bash
# Get workflow status
curl http://localhost:8081/api/v1/workflows/<workflow-id>/status
```

## 📝 File Format Requirements

The workflow supports these input formats:

### **CSV Files** (`.csv`)
```csv
domain,type,category
example.com,premium,technology  
test-site.org,standard,development
https://my-app.com,premium,application
```

### **TSV Files** (`.tsv`) 
```tsv
domain	type	category
example.com	premium	technology
test-site.org	standard	development  
```

### **Excel Files** (`.xlsx`, `.xls`)
- First column must contain domain names/labels
- Headers are automatically detected and skipped
- Supports multiple sheets (processes first sheet)

## 🔧 Customizing Examples

Create your own example by copying and modifying:

```bash
# Copy existing example
cp examples/domain-labels.example.json examples/my-custom.example.json

# Edit the parameters
{
  "file_uri": "s3://my-bucket/my-domains.csv",
  "tags": ["my-project", "batch-1"], 
  "created_by": "myemail@company.com",
  "description": "My custom domain processing job"
}

# Run your custom workflow
make start-domain-labels-workflow DOMAIN_INPUT=examples/my-custom.example.json
```

## 📈 Expected Results

After workflow completion, you'll see results like:

```json
{
  "processed_count": 10000,
  "saved_count": 8500, 
  "skipped_count": 1200,
  "error_count": 300,
  "labels": [
    {
      "id": 1,
      "label": "example",
      "original": "example.com",
      "tags": ["premium", "november-2024"],
      "created": true
    }
  ],
  "errors": ["Invalid domain: 'not-a-domain'"]
}
```

## ⚡ Performance Notes

- **Small files** (<1000 domains): Complete in seconds
- **Medium files** (1000-100K domains): Complete in minutes  
- **Large files** (100K+ domains): May take 10-30+ minutes
- **Massive files** (millions): Can run for hours with full fault tolerance

The workflow automatically handles:
- ✅ Progress tracking with heartbeats
- ✅ Automatic retries on failures  
- ✅ Database transaction safety
- ✅ Duplicate detection and merging
- ✅ Resume from failure points
