#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-pulltrace}"
KUBECONTEXT="${KUBECONTEXT:-d4b}"
KUBECTL="kubectl --context=${KUBECONTEXT}"

echo "=== Pulltrace E2E Test ==="

echo "1. Checking pods are running..."
${KUBECTL} -n "${NAMESPACE}" get pods
${KUBECTL} -n "${NAMESPACE}" wait --for=condition=Ready pod -l app.kubernetes.io/name=pulltrace --timeout=120s

echo "2. Checking server health..."
SERVER_POD=$(${KUBECTL} -n "${NAMESPACE}" get pod -l app.kubernetes.io/component=server -o jsonpath='{.items[0].metadata.name}')
${KUBECTL} -n "${NAMESPACE}" exec "${SERVER_POD}" -- wget -qO- http://localhost:8080/healthz
echo ""

echo "3. Checking API endpoint..."
${KUBECTL} -n "${NAMESPACE}" exec "${SERVER_POD}" -- wget -qO- http://localhost:8080/api/v1/pulls
echo ""

echo "4. Checking metrics endpoint..."
METRICS=$(${KUBECTL} -n "${NAMESPACE}" exec "${SERVER_POD}" -- wget -qO- http://localhost:9090/metrics 2>/dev/null | head -20)
echo "${METRICS}"

echo "5. Triggering a test pull (large image)..."
${KUBECTL} -n "${NAMESPACE}" run pulltrace-test \
  --image=docker.io/library/ubuntu:24.04 \
  --restart=Never \
  --command -- sleep 10 2>/dev/null || true

echo "Waiting 10s for pull to start..."
sleep 10

echo "6. Checking for pull events in server logs..."
${KUBECTL} -n "${NAMESPACE}" logs "${SERVER_POD}" --tail=20 | head -10

echo "7. Cleanup test pod..."
${KUBECTL} -n "${NAMESPACE}" delete pod pulltrace-test --ignore-not-found

echo ""
echo "=== E2E Test Complete ==="
