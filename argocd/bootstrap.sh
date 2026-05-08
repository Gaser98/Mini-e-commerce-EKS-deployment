#!/usr/bin/env bash
# One-time ArgoCD installation into the EKS cluster.
# Run this once after the cluster is provisioned; never needs to run again.
#
# Prerequisites:
#   - kubectl configured against the ecommerce-eks cluster
#   - cluster-admin permissions for the current IAM principal

set -euo pipefail

ARGOCD_VERSION="v2.13.3"

echo "==> Installing ArgoCD ${ARGOCD_VERSION}"
kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -n argocd \
  -f "https://raw.githubusercontent.com/argoproj/argo-cd/${ARGOCD_VERSION}/manifests/install.yaml"

echo "==> Waiting for ArgoCD server to be ready..."
kubectl rollout status deployment/argocd-server -n argocd --timeout=120s

echo "==> Applying Application manifests"
kubectl apply -f "$(dirname "$0")/staging.yaml"
kubectl apply -f "$(dirname "$0")/production.yaml"

echo "==> Retrieving initial admin password"
echo "    argocd admin initial-password -n argocd"
argocd admin initial-password -n argocd 2>/dev/null || \
  kubectl get secret argocd-initial-admin-secret -n argocd \
    -o jsonpath='{.data.password}' | base64 -d && echo

echo ""
echo "==> ArgoCD UI"
echo "    kubectl port-forward svc/argocd-server -n argocd 8080:443"
echo "    Then open: https://localhost:8080  (user: admin)"
echo ""
echo "==> To expose via LoadBalancer instead:"
echo "    kubectl patch svc argocd-server -n argocd -p '{\"spec\":{\"type\":\"LoadBalancer\"}}'"
