# 🛡️ Infrastructure - Kubernetes & Terraform

## Структура

```
infra/
├── k8s/                   # Kubernetes manifests
│   ├── base/              # Base resources
│   │   ├── namespace.yaml
│   │   ├── configmap.yaml
│   │   └── secrets.yaml
│   ├── overlays/
│   │   ├── dev/           # Development environment
│   │   ├── staging/       # Staging environment
│   │   └── production/    # Production environment
│   ├── apps/
│   │   ├── backend/       # Backend deployment
│   │   ├── postgres/      # PostgreSQL StatefulSet
│   │   ├── redis/         # Redis cluster
│   │   └── monitoring/    # Prometheus, Grafana
│   └── ingress/           # Ingress configuration
├── terraform/             # IaC
│   ├── modules/
│   │   ├── gke/           # Google Kubernetes Engine
│   │   ├── rds/           # Managed PostgreSQL
│   │   ├── redis/         # Managed Redis
│   │   └── networking/    # VPC, subnets, firewall
│   ├── environments/
│   │   ├── dev/
│   │   ├── staging/
│   │   └── production/
│   └── state/             # Remote state (GCS/S3)
└── helm/                  # Helm charts (optional)
```

## Kubernetes Deployment

### Backend Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: messenger-backend
  namespace: messenger
spec:
  replicas: 3
  selector:
    matchLabels:
      app: backend
  template:
    metadata:
      labels:
        app: backend
    spec:
      containers:
      - name: backend
        image: messenger-backend:latest
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: host
        resources:
          requests:
            cpu: 100m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 512Mi
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

### Horizontal Pod Autoscaler
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: backend-hpa
  namespace: messenger
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: messenger-backend
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### Ingress (nginx)
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: messenger-ingress
  namespace: messenger
  annotations:
    nginx.ingress.kubernetes.io/websocket-services: "backend"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
spec:
  tls:
  - hosts:
    - api.messenger.app
    secretName: tls-secret
  rules:
  - host: api.messenger.app
    http:
      paths:
      - path: /ws
        pathType: Prefix
        backend:
          service:
            name: backend
            port:
              number: 80
      - path: /
        pathType: Prefix
        backend:
          service:
            name: backend
            port:
              number: 80
```

## Terraform Configuration

### GKE Cluster
```hcl
module "gke" {
  source = "./modules/gke"
  
  project_id  = var.project_id
  cluster_name = "messenger-cluster"
  region      = var.region
  
  node_pools = [
    {
      name         = "default-pool"
      machine_type = "e2-standard-4"
      min_count    = 3
      max_count    = 10
    }
  ]
}
```

### Cloud SQL (PostgreSQL)
```hcl
module "database" {
  source = "./modules/rds"
  
  project_id    = var.project_id
  instance_name = "messenger-db"
  database_version = "POSTGRES_15"
  
  tier = "db-custom-4-15360"
  
  backup_configuration = {
    enabled                        = true
    start_time                     = "02:00"
    point_in_time_recovery_enabled = true
  }
}
```

## Monitoring Stack

### Prometheus
```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: backend-monitor
  namespace: messenger
spec:
  selector:
    matchLabels:
      app: backend
  endpoints:
  - port: http
    path: /metrics
    interval: 15s
```

### Grafana Dashboards
- Request rate / latency
- Error rates
- WebSocket connections
- Database connection pool
- Memory / CPU usage

## CI/CD Pipeline

```yaml
# .github/workflows/deploy.yml
name: Deploy

on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Build Docker image
      run: docker build -t messenger-backend:${{ github.sha }} ./backend
    
    - name: Push to registry
      run: docker push ${{ secrets.REGISTRY }}/messenger-backend:${{ github.sha }}
    
    - name: Deploy to Kubernetes
      run: |
        kubectl set image deployment/messenger-backend \
          backend=${{ secrets.REGISTRY }}/messenger-backend:${{ github.sha }}
        kubectl rollout status deployment/messenger-backend
```

## Security

- Network Policies (zero-trust)
- Pod Security Standards (restricted)
- Secrets encryption at rest
- mTLS between services
- Regular security scans (Trivy)

## Cost Optimization

- Spot instances for non-critical workloads
- Auto-scaling based on load
- Resource quotas and limits
- Scheduled scaling for off-peak hours
