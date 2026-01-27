# Echo Backend - Terraform Infrastructure

Terraform configuration to provision complete Echo Backend infrastructure on Google Cloud Platform with Cloudflare integration.

## What Gets Provisioned

- **GKE Cluster** - Kubernetes cluster with auto-scaling
- **Cloud SQL** - PostgreSQL database with regional high availability
- **Redis** - Memorystore for caching
- **VPC Network** - Private networking with subnets
- **Load Balancer** - Global static IP
- **Cloudflare DNS** (optional) - DNS records for API and WebSocket endpoints

---

## Prerequisites

### 1. Install Required Tools

```bash
# Install Terraform
brew install terraform

# Install Google Cloud SDK
brew install --cask google-cloud-sdk

# Install kubectl
brew install kubectl
```

### 2. Authenticate with Google Cloud

```bash
# Login to GCP
gcloud auth login

# Set your project
gcloud config set project YOUR_PROJECT_ID

# Application default credentials for Terraform
gcloud auth application-default login
```

### 3. Enable Required GCP APIs

```bash
gcloud services enable \
  compute.googleapis.com \
  container.googleapis.com \
  sqladmin.googleapis.com \
  redis.googleapis.com \
  servicenetworking.googleapis.com \
  cloudresourcemanager.googleapis.com
```

### 4. Cloudflare Setup (Optional)

Cloudflare is **optional**. If you don't use Cloudflare:
- Set `enable_cloudflare = false` in your `terraform.tfvars`
- Manually configure DNS to point to the load balancer IP after deployment
- Skip the Cloudflare credential steps below

If you want to use Cloudflare for DNS:
1. **Get API Token:**
   - Sign up at https://cloudflare.com if you don't have an account
   - Add your domain to Cloudflare
   - Go to: Profile → API Tokens → Create Token
   - Use "Edit zone DNS" template
   - Copy the token

2. **Get Zone ID:**
   - Go to your domain's dashboard in Cloudflare
   - Find "Zone ID" in the right sidebar under API section
   - Copy the Zone ID

3. **Set in terraform.tfvars:**
   ```hcl
   enable_cloudflare    = true
   cloudflare_api_token = "your-token-here"
   cloudflare_zone_id   = "your-zone-id-here"
   ```

---

## Step-by-Step Deployment

### Step 1: Create State Storage Bucket

```bash
# Create GCS bucket for Terraform state
gsutil mb -p YOUR_PROJECT_ID gs://echo-terraform-state

# Enable versioning
gsutil versioning set on gs://echo-terraform-state
```

### Step 2: Configure Variables

Choose your environment (dev or prod):

```bash
cd infra/terraform/environments/dev  # or prod
```

Copy and edit the configuration:

```bash
cp terraform.tfvars.example terraform.tfvars
nano terraform.tfvars
```

**Required values in `terraform.tfvars`:**

```hcl
# GCP Configuration
project_id = "your-gcp-project-id"
region     = "us-central1"

# Cloudflare (OPTIONAL - set to false if not using Cloudflare)
enable_cloudflare = false

# If enable_cloudflare = true, also add:
# cloudflare_api_token = "your-cloudflare-api-token-here"
# cloudflare_zone_id   = "your-cloudflare-zone-id-here"

# Other values are already set in the example file
```

### Step 3: Initialize Terraform

```bash
cd ../..  # Return to infra/terraform directory

# Initialize Terraform and download providers
terraform init
```

### Step 4: Plan Infrastructure

```bash
# Preview what will be created (dev example)
terraform plan -var-file="environments/dev/terraform.tfvars"
```

Review the plan to ensure everything looks correct.

### Step 5: Deploy Infrastructure

```bash
# Create the infrastructure
terraform apply -var-file="environments/dev/terraform.tfvars"

# Type 'yes' when prompted
```

⏱️ **This takes 15-25 minutes** (GKE cluster creation is the longest step)

### Step 6: Get Outputs

```bash
# View all outputs
terraform output

# Get database password (sensitive)
terraform output postgres_password

# Get connection details
terraform output postgres_connection_name
terraform output redis_host
```

### Step 7: Connect to GKE Cluster

```bash
# Get kubectl credentials
gcloud container clusters get-credentials CLUSTER_NAME \
  --region REGION \
  --project PROJECT_ID

# Or use the generated command
terraform output gcloud_get_credentials_command
$(terraform output -raw gcloud_get_credentials_command)

# Verify connection
kubectl get nodes
```

### Step 8: Configure DNS (if not using Cloudflare)

If you set `enable_cloudflare = false`, manually configure your DNS:

```bash
# Get the load balancer IP
terraform output load_balancer_ip

# Add these DNS records in your DNS provider:
# A record: api.yourdomain.com  → load_balancer_ip
# A record: ws.yourdomain.com   → load_balancer_ip
```

### Step 9: Deploy Applications

```bash
cd ../kubernetes

# Create namespace
kubectl apply -f namespaces/

# Create configmaps and secrets
kubectl apply -f configmaps/
kubectl apply -f secrets/

# Deploy services
kubectl apply -f deployments/

# Setup ingress
kubectl apply -f ingress/
```

---

## Environment Management

### Development Environment

```bash
# Plan
terraform plan -var-file="environments/dev/terraform.tfvars"

# Apply
terraform apply -var-file="environments/dev/terraform.tfvars"

# Destroy
terraform destroy -var-file="environments/dev/terraform.tfvars"
```

### Production Environment

```bash
# Plan
terraform plan -var-file="environments/prod/terraform.tfvars"

# Apply
terraform apply -var-file="environments/prod/terraform.tfvars"
```

---

## Common Operations

### View Current Infrastructure

```bash
# List all resources
terraform state list

# View specific resource details
terraform state show module.gcp.google_container_cluster.primary

# Show outputs
terraform output
```

### Update Infrastructure

1. Modify values in `terraform.tfvars`
2. Run plan to preview changes:
   ```bash
   terraform plan -var-file="environments/dev/terraform.tfvars"
   ```
3. Apply changes:
   ```bash
   terraform apply -var-file="environments/dev/terraform.tfvars"
   ```

### Connect to Cloud SQL

```bash
# Get connection details
terraform output postgres_connection_name

# Using Cloud SQL Proxy
cloud_sql_proxy -instances=$(terraform output -raw postgres_connection_name)=tcp:5432

# In another terminal, connect with psql
psql "host=127.0.0.1 port=5432 dbname=echo_db user=echo_user"
# Enter password from: terraform output postgres_password
```

### Scale GKE Nodes

```bash
# Scale up/down
gcloud container clusters resize CLUSTER_NAME \
  --num-nodes=5 \
  --region=REGION

# Or update terraform.tfvars and apply
gke_node_count = 5
terraform apply -var-file="environments/dev/terraform.tfvars"
```

---

## Troubleshooting

### Issue: API Not Enabled

**Error:** `Error 403: ... API has not been used in project`

**Solution:**
```bash
# Enable the specific API mentioned in the error
gcloud services enable SERVICE_NAME.googleapis.com
```

### Issue: State Lock

**Error:** `Error: Error acquiring the state lock`

**Solution:**
```bash
# Force unlock (use Lock ID from error message)
terraform force-unlock LOCK_ID
```

### Issue: Backend Configuration Changed

**Error:** `Backend configuration changed`

**Solution:**
```bash
terraform init -reconfigure
```

### Issue: Insufficient Quota

**Error:** `Quota 'CPUS' exceeded`

**Solution:**
- Go to GCP Console → IAM & Admin → Quotas
- Request quota increase for the specific resource

### Issue: VPC Peering Error

**Error:** `Error creating Cloud SQL instance: Invalid network`

**Solution:**
- Ensure the VPC private service connection was created
- This is now handled automatically by the configuration
- Check: `terraform state show module.gcp.google_service_networking_connection.private_vpc_connection`

---

## File Structure

```
terraform/
├── main.tf                    # Main configuration
├── variables.tf               # Variable definitions
├── outputs.tf                 # Output definitions
├── .gitignore                 # Git ignore rules
├── README.md                  # This file
├── environments/
│   ├── dev/
│   │   ├── terraform.tfvars.example
│   │   └── terraform.tfvars (create this)
│   └── prod/
│       ├── terraform.tfvars.example
│       └── terraform.tfvars (create this)
└── modules/
    ├── gcp/                   # GKE & Networking
    ├── database/              # Cloud SQL & Redis
    └── cloudflare/            # DNS & Security
```

---

## Cost Estimates

### Development (~$150-200/month)
- GKE: 2 nodes (e2-standard-2) - ~$60
- Cloud SQL: db-custom-2-8192 - ~$80
- Redis: 2GB Basic - ~$30
- Networking & LB - ~$30

### Production (~$500-700/month)
- GKE: 5 nodes (e2-standard-4) - ~$250
- Cloud SQL: db-custom-8-32768 - ~$300
- Redis: 10GB Standard HA - ~$100
- Networking & LB - ~$50

---

## Security Notes

1. **Never commit `.tfvars` files** - They contain sensitive data (already in `.gitignore`)
2. **Rotate credentials regularly** - Database passwords, API tokens
3. **Use least privilege** - Grant minimal required GCP permissions
4. **Enable deletion protection** - Already enabled for Cloud SQL
5. **Store state securely** - GCS bucket with versioning enabled

---

## Next Steps After Deployment

1. **Run database migrations:**
   ```bash
   psql -h POSTGRES_IP -U echo_user -d echo_db -f database/schemas/user-schema.sql
   psql -h POSTGRES_IP -U echo_user -d echo_db -f database/schemas/message-schema.sql
   # ... other schemas
   ```

2. **Verify DNS propagation:**
   ```bash
   dig api.yourdomain.com
   dig ws.yourdomain.com
   ```

3. **Deploy your services to Kubernetes**

4. **Setup monitoring and logging**

---

## Useful Commands

```bash
# Terraform
terraform fmt                   # Format code
terraform validate              # Validate syntax
terraform plan                  # Preview changes
terraform apply                 # Apply changes
terraform destroy              # Destroy infrastructure
terraform output                # Show outputs
terraform state list            # List resources

# GCP
gcloud container clusters list  # List GKE clusters
gcloud sql instances list       # List Cloud SQL instances
gcloud compute addresses list   # List static IPs

# Kubernetes
kubectl get nodes               # List cluster nodes
kubectl get pods -A             # List all pods
kubectl get svc -A              # List all services
kubectl describe node NODE_NAME # Node details
```

---

## Support

For issues or questions:

1. Check the [Troubleshooting](#troubleshooting) section
2. Review Terraform logs: `TF_LOG=DEBUG terraform apply`
3. Check GCP logs: `gcloud logging read`
4. Review provider documentation:
   - [Terraform Google Provider](https://registry.terraform.io/providers/hashicorp/google/latest/docs)
   - [Terraform Cloudflare Provider](https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs)

---

**Last Updated:** 2026-01-27
