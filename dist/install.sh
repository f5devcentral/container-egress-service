#!/usr/bin/env bash
set -euo pipefail

BIGIP_URL=${BIGIP_URL:-}               # IP address of Big-IP server
BIGIP_USERNAME=${BIGIP_USERNAME:-}     # BigIP username
BIGIP_PASSWORD=${BIGIP_PASSWORD:-}     # BigIP password
BIGIP_INSECURE=${BIGIP_INSECURE:-true} # ignore Big-IP TLS error

GATEWAY=${GATEWAY:-} # gateway address

BIGIP_URL="192.168.50.75"
BIGIP_USERNAME="admin"
BIGIP_PASSWORD="nihao666"
GATEWAY="192.168.1.1"

K8S_NAMESPACE=${K8S_NAMESPACE:-kube-system} # namespace in which the controller will be deployed

echo "[Step 1] Create Secret"
kubectl -n $K8S_NAMESPACE create secret generic --from-literal "username=$BIGIP_USERNAME" --from-literal "password=$BIGIP_PASSWORD" f5-bigip-creds
echo "-------------------------------"
echo ""

echo "[Step 2] Apply CRD"
cat << EOF | kubectl apply -f -
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: externalservices.kubeovn.io
spec:
  scope: Cluster
  group: kubeovn.io
  names:
    kind: ExternalService
    singular: externalservice
    plural: externalservices
    shortNames:
      - exsvc
  versions:
    - name: v1alpha1
      served: true
      storage: true
      additionalPrinterColumns:
        - name: Addresses
          type: string
          jsonPath: .spec.addresses
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                addresses:
                  type: array
                  items:
                    type: string
                    format: "ip"
                ports:
                  type: array
                  items:
                    type: object
                    properties:
                      name:
                        type: string
                      protocol:
                        type: string
                      port:
                        type: string
                        pattern: "^[0-9]+(-[0-9]+)?(,[0-9]+(-[0-9]+)?)*$"
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: f5firewallrules.kubeovn.io
spec:
  scope: Cluster
  group: kubeovn.io
  names:
    kind: F5FirewallRule
    singular: f5firewallrule
    plural: f5firewallrules
    shortNames:
      - f5fwrule
  versions:
    - name: v1alpha1
      served: true
      storage: true
      additionalPrinterColumns:
        - name: Action
          type: string
          jsonPath: .spec.action
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                action:
                  type: string
                  enum:
                    - accept
                    - drop
                    - accept-decisively
                    - reject
                services:
                  type: array
                  items:
                    type: string
                externalServices:
                  type: array
                  items:
                    type: string
EOF
echo "-------------------------------"
echo ""

echo "[Step 3] Apply RBAC"
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: f5-as3-ctlr
  namespace: $K8S_NAMESPACE
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: f5-as3-ctlr
rules:
  - apiGroups:
      - ""
    resources:
      - endpoints
    verbs:
      - get
      - watch
      - list
  - apiGroups:
      - ""
    resources:
      - configmaps
    resourceNames:
      - f5-as3-ctlr
    verbs:
      - get
      - update
  - apiGroups:
      - ""
    resources:
      - events
    verbs:
      - create
      - patch
      - update
  - apiGroups:
      - kubeovn.io
    resources:
      - externalservices
      - f5servicefirewallrules
      - f5globalfirewallrules
      - f5namespacefirewallrules
    verbs:
      - get
      - watch
      - list
      - update
      - patch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: f5-as3-ctlr
subjects:
  - kind: ServiceAccount
    name: f5-as3-ctlr
    namespace: $K8S_NAMESPACE
roleRef:
  kind: ClusterRole
  name: f5-as3-ctlr
  apiGroup: rbac.authorization.k8s.io
EOF
echo "-------------------------------"
echo ""

echo "[Step 4] Create ConfigMap"
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: f5-as3-ctlr
  namespace: $K8S_NAMESPACE
data:
  initialized: "false"
EOF
echo "-------------------------------"
echo ""

echo "[Step 5] Apply AS3 Controller"
cat << EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: f5-as3-ctlr
  namespace: $K8S_NAMESPACE
spec:
  replicas: 1
  selector:
    matchLabels:
      app: f5-as3-ctlr
  template:
    metadata:
      labels:
        app: f5-as3-ctlr
    spec:
      serviceAccountName: f5-as3-ctlr
      containers:
        - name: f5-as3-ctlr
          image: kubeovn/f5-as3-ctlr:0.1.0
          imagePullPolicy: IfNotPresent
          command:
            - /f5-as3-ctlr
            - --bigip-url=$BIGIP_URL
            - --bigip-insecure=$BIGIP_INSECURE
            - --bigip-creds-dir=/f5-bigip-creds
            - --gateway=$GATEWAY
          volumeMounts:
            - name: f5-bigip-creds
              mountPath: "/f5-bigip-creds"
              readOnly: true
      volumes:
        - name: f5-bigip-creds
          secret:
            secretName: f5-bigip-creds
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      terminationGracePeriodSeconds: 30
EOF
echo "-------------------------------"
echo ""

echo "[Step 5] Wait AS3 Controller to Be Ready"
kubectl -n $K8S_NAMESPACE wait pod --for=condition=Ready -l app=f5-as3-ctlr
