#!/bin/bash
# setup-observability-dev.sh - Local observability setup for Tekton Chains development
# This script sets up a Kind cluster with Tekton Chains, Prometheus, and Jaeger for observability.
# Services are accessed via port-forward for simplicity.
# It assumes you have `kind`, `kubectl`, and `ko` installed and configured.

set -euo pipefail

# Configuration
: "${KO_DOCKER_REPO:=kind.local}" # Local registry for development
: "${KIND_CLUSTER_NAME:=chains-dev}"
: "${PIPELINES_VERSION:=v1.10.2}" # Must match the version in go.mod

wait_for_deploy() {
  local ns="$1"
  local name="$2"
  echo "Waiting for deployment $name in namespace $ns..."
  for i in {1..60}; do
    if kubectl -n "$ns" get deploy "$name" >/dev/null 2>&1; then
      break
    fi
    sleep 2
  done
  kubectl -n "$ns" rollout status deploy/"$name" --timeout=300s
}

setup_port_forwards() {
  echo "Setting up port forwards..."
  
  # Kill any existing port-forwards for the namespaces used by this script
  pkill -f "kubectl.*port-forward.*-n monitoring" || true
  pkill -f "kubectl.*port-forward.*-n observability-system" || true
  pkill -f "kubectl.*port-forward.*-n tekton-chains" || true
  sleep 2
  
  # Setup port forwards in background
  kubectl port-forward -n monitoring svc/prometheus 9091:9090 > /dev/null 2>&1 &
  kubectl port-forward -n observability-system svc/jaeger 16686:16686 > /dev/null 2>&1 &
  kubectl port-forward -n tekton-chains svc/tekton-chains-metrics 9090:9090 > /dev/null 2>&1 &
  
  echo "Port forwards started in background"
}

export_metrics_to_csv() {
  echo "Exporting metrics to CSV files..."
  
  # Wait a moment for port forwards to be ready
  sleep 3
  
  # Define components with their ports and names
  declare -A components=(
    ["controller"]="9090"
  )
  
  for component in "${!components[@]}"; do
    port="${components[$component]}"
    output_file="chains-${component}-metrics.csv"
    
    # Fetch metrics and parse them
    curl -s "http://localhost:${port}/metrics" > /tmp/metrics_${component}.txt
    
    # Create CSV with headers
    echo "metric_name,type,help" > "${output_file}"
    
    # Parse metrics using awk
    awk '
      /^# HELP / {
        help_name = $3
        help_text = ""
        for (i=4; i<=NF; i++) help_text = help_text (i==4 ? "" : " ") $i
        gsub(/"/, "\"\"", help_text)  # Escape quotes
        help[help_name] = help_text
      }
      /^# TYPE / {
        type_name = $3
        type_value = $4
        if (type_name in help) {
          printf "\"%s\",\"%s\",\"%s\"\n", type_name, type_value, help[type_name]
        } else {
          printf "\"%s\",\"%s\",\"\"\n", type_name, type_value
        }
      }
    ' /tmp/metrics_${component}.txt >> "${output_file}"
    
  done
  
  # Cleanup temp files
  rm -f /tmp/metrics_*.txt
  
  echo ""
  echo "CSV files created:"
  ls -lh chains-*-metrics.csv
}

echo "Setting up observability stack..."

# Create Kind cluster
echo "Creating Kind cluster..."
kind create cluster --name "${KIND_CLUSTER_NAME}" --config - << 'EOF'
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  image: kindest/node:v1.32.0
EOF

# Set kubectl context to the new cluster
kubectl config use-context "kind-${KIND_CLUSTER_NAME}"

# Install Tekton Pipelines (required by Chains for TaskRun/PipelineRun CRDs)
echo "Installing Tekton Pipelines ${PIPELINES_VERSION}..."
kubectl apply --filename \
  "https://infra.tekton.dev/tekton-releases/pipeline/previous/${PIPELINES_VERSION}/release.yaml"

wait_for_deploy tekton-pipelines tekton-pipelines-controller
wait_for_deploy tekton-pipelines tekton-pipelines-webhook

# Build and deploy Tekton Chains from source
echo "Building and deploying Tekton Chains from source..."
export KO_DOCKER_REPO
export KIND_CLUSTER_NAME
ko apply -f config/

wait_for_deploy tekton-chains tekton-chains-controller

# Install Prometheus
echo "Installing Prometheus..."
kubectl apply -f - << EOF
apiVersion: v1
kind: Namespace
metadata:
  name: monitoring
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: monitoring
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
      metric_name_validation_scheme: legacy
    scrape_configs:
    - job_name: 'tekton-chains-controller'
      metric_name_escaping_scheme: underscores
      static_configs:
      - targets: ['tekton-chains-metrics.tekton-chains.svc.cluster.local:9090']
    - job_name: 'kubernetes-pods'
      metric_name_escaping_scheme: underscores
      kubernetes_sd_configs:
      - role: pod
        namespaces:
          names: [tekton-chains]
      relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: prometheus
  template:
    metadata:
      labels:
        app: prometheus
    spec:
      containers:
      - name: prometheus
        image: prom/prometheus:latest
        ports:
        - containerPort: 9090
        volumeMounts:
        - name: config
          mountPath: /etc/prometheus
        args:
        - '--config.file=/etc/prometheus/prometheus.yml'
        - '--storage.tsdb.path=/prometheus'
        - '--web.console.libraries=/etc/prometheus/console_libraries'
        - '--web.console.templates=/etc/prometheus/consoles'
      volumes:
      - name: config
        configMap:
          name: prometheus-config
---
apiVersion: v1
kind: Service
metadata:
  name: prometheus
  namespace: monitoring
spec:
  selector:
    app: prometheus
  ports:
  - port: 9090
    targetPort: 9090
  type: ClusterIP
EOF

wait_for_deploy monitoring prometheus

# Install Jaeger
echo "Installing Jaeger..."
kubectl apply -f - << EOF
apiVersion: v1
kind: Namespace
metadata:
  name: observability-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jaeger
  namespace: observability-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: jaeger
  template:
    metadata:
      labels:
        app: jaeger
    spec:
      containers:
      - name: jaeger
        image: jaegertracing/all-in-one:latest
        ports:
        - containerPort: 16686
        - containerPort: 14268
        env:
        - name: COLLECTOR_OTLP_ENABLED
          value: "true"
---
apiVersion: v1
kind: Service
metadata:
  name: jaeger
  namespace: observability-system
spec:
  selector:
    app: jaeger
  ports:
  - name: ui
    port: 16686
    targetPort: 16686
  - name: collector
    port: 14268
    targetPort: 14268
  type: ClusterIP
EOF

wait_for_deploy observability-system jaeger

# Update tekton-chains-config-observability to enable Prometheus metrics
echo "Configuring Chains observability..."
kubectl patch configmap tekton-chains-config-observability -n tekton-chains --type merge -p '{
  "data": {
    "metrics-protocol": "prometheus"
  }
}'

# Restart chains controller to pick up configuration
echo "Restarting Tekton Chains controller to apply observability config..."
kubectl rollout restart deployment/tekton-chains-controller -n tekton-chains
kubectl rollout status deployment/tekton-chains-controller -n tekton-chains --timeout=300s

# Setup port forwards
setup_port_forwards

# Export metrics to CSV
export_metrics_to_csv

echo "Setup complete!"
echo ""
echo "Access URLs (via port-forward):"
echo "  Prometheus:              http://localhost:9091"
echo "  Jaeger:                  http://localhost:16686"
echo "  Chains metrics endpoint: http://localhost:9090/metrics"
echo ""
echo "To stop port-forwards: pkill -f 'kubectl.*port-forward'"