#!/usr/bin/env bash
set -euo pipefail

CES_NAMESPACE=${CES_NAMESPACE:-kube-system} # namespace in which the controller will be deployed
CES_DEPLOMENT_NAME=${CES_DEPLOMENT_NAME:-ces-controller}

echo "[Step 1] Delete CES Controller"
kubectl -n $CES_NAMESPACE delete --ignore-not-found deployment $CES_DEPLOMENT_NAME
echo "-------------------------------"
echo ""

echo "[Step 2] Delete RBAC"
kubectl delete --ignore-not-found clusterrolebinding ces-controller
kubectl delete --ignore-not-found clusterrole ces-controller
kubectl -n $CES_NAMESPACE delete --ignore-not-found sa ces-controller
echo "-------------------------------"
echo ""

echo "[Step 3] Delete CRD"
kubectl delete --ignore-not-found crd externalservices.kubeovn.io
kubectl delete --ignore-not-found crd clusteregressrules.kubeovn.io
kubectl delete --ignore-not-found crd namespaceegressrules.kubeovn.io
kubectl delete --ignore-not-found crd serviceegressrules.kubeovn.io
kubectl delete --ignore-not-found crd externaliprules.bigip.io
echo "-------------------------------"
echo ""

echo "[Step 4] Delete Secret"
kubectl -n $CES_NAMESPACE delete --ignore-not-found secret bigip-creds
echo "-------------------------------"
echo ""

echo "[Step 5] Delete ConfigMap"
kubectl -n $CES_NAMESPACE delete --ignore-not-found cm ces-controller-configmap
echo "-------------------------------"
echo ""
