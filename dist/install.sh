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
  scope: Namespaced
  group: kubeovn.io
  names:
    kind: ExternalService
    listKind: ExternalServiceList
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
  name: clusteregressrules.kubeovn.io
spec:
  scope: Cluster
  group: kubeovn.io
  names:
    kind: ClusterEgressRule
    listKind: ClusterEgressRuleList
    plural: clusteregressrules
    singular: clusteregressrule
    shortNames:
      - cgr
  versions:
    - name: v1alpha1
      served: true
      storage: true
      additionalPrinterColumns:
        - name: Action
          type: string
          jsonPath: .spec.action
        - name: Status
          type: string
          jsonPath: .status.phase
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              required:
                - action
                - externalServices
              properties:
                action:
                  type: string
                  enum:
                    - accept
                    - drop
                    - accept-decisively
                    - reject
                externalServices:
                  type: array
                  items:
                    type: string
            status:
              properties:
                phase:
                  type: string
              type: object
      subresources:
        status: {}
---

---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: namespaceegressrules.kubeovn.io
spec:
  scope: Namespaced
  group: kubeovn.io
  names:
    kind: NamespaceEgressRule
    listKind: NamespaceEgressRuleList
    plural: namespaceegressrules
    singular: namespaceegressrule
    shortNames:
      - nsgr
  versions:
    - name: v1alpha1
      served: true
      storage: true
      additionalPrinterColumns:
        - name: Action
          type: string
          jsonPath: .spec.action
        - name: Status
          type: string
          jsonPath: .status.phase
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              required:
                - action
                - externalServices
              properties:
                action:
                  type: string
                  enum:
                    - accept
                    - drop
                    - accept-decisively
                    - reject
                externalServices:
                  type: array
                  items:
                    type: string
            status:
              properties:
                phase:
                  type: string
              type: object
      subresources:
        status: {}
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: serviceegressrules.kubeovn.io
spec:
  scope: Namespaced
  group: kubeovn.io
  names:
    kind: ServiceEgressRule
    listKind: ServiceEgressRuleList
    plural: serviceegressrules
    singular: serviceegressrule
    shortNames:
      - svcgr
  versions:
    - name: v1alpha1
      served: true
      storage: true
      additionalPrinterColumns:
        - name: Action
          type: string
          jsonPath: .spec.action
        - name: Status
          type: string
          jsonPath: .status.phase
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              required:
                - action
                - externalServices
                - service
              properties:
                action:
                  type: string
                  enum:
                    - accept
                    - drop
                    - accept-decisively
                    - reject
                service:
                  type: string
                externalServices:
                  type: array
                  items:
                    type: string
            status:
              properties:
                phase:
                  type: string
              type: object
      subresources:
        status: {}
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
      - namespaces
    verbs:
      - get
      - watch
      - list
  - apiGroups:
      - ""
    resources:
      - configmaps
    resourceNames:
      - ces-controller-configmap
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
      - clusteregressrules
      - namespaceegressrules
      - serviceegressrules
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
  name: ces-controller
subjects:
  - kind: ServiceAccount
    name: ces-controller
    namespace: $K8S_NAMESPACE
roleRef:
  kind: ClusterRole
  name: ces-controller
  apiGroup: rbac.authorization.k8s.io
EOF
echo "-------------------------------"
echo ""

echo "[Step 4] Create ConfigMap"
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: ces-controller-configmap
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
  name: ces-controller
  namespace: $K8S_NAMESPACE
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ces-controller
  template:
    metadata:
      labels:
        app: ces-controller
    spec:
      serviceAccountName: ces-controller
      containers:
        - name: ces-controller
          image: kubeovn/ces-controller:0.1.0
          imagePullPolicy: IfNotPresent
          command:
            - /f5-as3-ctlr
            - --bigip-url=$BIGIP_URL
            - --bigip-insecure=$BIGIP_INSECURE
            - --bigip-creds-dir=/bigip-creds
            - --gateway=$GATEWAY
          volumeMounts:
            - name: bigip-creds
              mountPath: "/bigip-creds"
              readOnly: true
      volumes:
        - name: bigip-creds
          secret:
            secretName: bigip-creds
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      terminationGracePeriodSeconds: 30
EOF
echo "-------------------------------"
echo ""

echo "[Step 5] Wait CES Controller to Be Ready"
kubectl -n $K8S_NAMESPACE wait pod --for=condition=Ready -l app=ces-controller
