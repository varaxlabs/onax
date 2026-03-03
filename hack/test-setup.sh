#!/bin/bash
# test-setup.sh - Create a kind cluster for local testing
set -euo pipefail

CLUSTER_NAME="cronjob-monitor-test"

create() {
    echo "Creating kind cluster: ${CLUSTER_NAME}"

    kind create cluster --name "${CLUSTER_NAME}" --config - <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
  - role: worker
EOF

    echo "Installing Prometheus stack..."
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts 2>/dev/null || true
    helm repo update

    helm install prometheus prometheus-community/kube-prometheus-stack \
        --namespace monitoring \
        --create-namespace \
        --set grafana.enabled=true \
        --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
        --wait

    echo ""
    echo "Kind cluster '${CLUSTER_NAME}' created with Prometheus stack."
    echo ""
    echo "To deploy the operator:"
    echo "  make docker-build"
    echo "  kind load docker-image ghcr.io/kubeshield/k8s-cronjob-monitor:latest --name ${CLUSTER_NAME}"
    echo "  make deploy"
    echo ""
    echo "To deploy test CronJobs:"
    echo "  kubectl apply -f examples/basic-cronjob.yaml"
    echo ""
    echo "To access Grafana:"
    echo "  kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80"
    echo "  Open http://localhost:3000 (admin/prom-operator)"
}

delete() {
    echo "Deleting kind cluster: ${CLUSTER_NAME}"
    kind delete cluster --name "${CLUSTER_NAME}"
}

case "${1:-}" in
    create)
        create
        ;;
    delete)
        delete
        ;;
    *)
        echo "Usage: $0 {create|delete}"
        exit 1
        ;;
esac
