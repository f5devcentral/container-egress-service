#!/usr/bin/env bash
set -euo pipefail

K8S_NAMESPACE=${K8S_NAMESPACE:-kube-system} # namespace in which the controller will be deployed 

echo "[Step 1] Delete CES Controller"
kubectl -n $K8S_NAMESPACE delete --ignore-not-found deployment ces-controller
echo "-------------------------------"
echo ""

echo "[Step 2] Delete RBAC"
kubectl delete --ignore-not-found clusterrolebinding ces-controller
kubectl delete --ignore-not-found clusterrole ces-controller
kubectl -n $K8S_NAMESPACE delete --ignore-not-found sa ces-controller
echo "-------------------------------"
echo ""

echo "[Step 3] Delete CRD"
kubectl delete --ignore-not-found crd f5firewallrules.kubeovn.io
kubectl delete --ignore-not-found crd externalservices.kubeovn.io
echo "-------------------------------"
echo ""

echo "[Step 4] Delete Secret"
kubectl -n $K8S_NAMESPACE delete --ignore-not-found secret bigip-creds
echo "-------------------------------"
echo ""

echo "[Step 5] Delete ConfigMap"
kubectl -n $K8S_NAMESPACE delete --ignore-not-found ces-controller-configmap
echo "-------------------------------"
echo ""
