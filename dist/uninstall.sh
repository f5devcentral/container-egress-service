#!/usr/bin/env bash
set -euo pipefail

K8S_NAMESPACE=${K8S_NAMESPACE:-kube-system} # namespace in which the controller will be deployed 

echo "[Step 1] Delete AS3 Controller"
kubectl -n $K8S_NAMESPACE delete --ignore-not-found deployment f5-as3-ctlr
echo "-------------------------------"
echo ""

echo "[Step 2] Delete RBAC"
kubectl delete --ignore-not-found clusterrolebinding f5-as3-ctlr
kubectl delete --ignore-not-found clusterrole f5-as3-ctlr
kubectl -n $K8S_NAMESPACE delete --ignore-not-found sa f5-as3-ctlr
echo "-------------------------------"
echo ""

echo "[Step 3] Delete CRD"
kubectl delete --ignore-not-found crd f5firewallrules.kubeovn.io
kubectl delete --ignore-not-found crd externalservices.kubeovn.io
echo "-------------------------------"
echo ""

echo "[Step 4] Delete Secret"
kubectl -n $K8S_NAMESPACE delete --ignore-not-found secret f5-bigip-creds
echo "-------------------------------"
echo ""

echo "[Step 5] Delete ConfigMap"
kubectl -n $K8S_NAMESPACE delete --ignore-not-found cm f5-bigip-ctlr
echo "-------------------------------"
echo ""
