#!/usr/bin/env bash
set -euo pipefail

BIGIP_URL=${BIGIP_URL:-}               # IP address of Big-IP server
BIGIP_USERNAME=${BIGIP_USERNAME:-}     # BigIP username
BIGIP_PASSWORD=${BIGIP_PASSWORD:-}     # BigIP password
BIGIP_INSECURE=${BIGIP_INSECURE:-true} # ignore Big-IP TLS error


BIGIP_URL="192.168.50.75"
BIGIP_USERNAME="admin"
BIGIP_PASSWORD="nihao666"

K8S_NAMESPACE=${K8S_NAMESPACE:-kube-system} # namespace in which the controller will be deployed

echo "[Step 1] Create Secret"
kubectl -n $K8S_NAMESPACE create secret generic --from-literal "username=$BIGIP_USERNAME" --from-literal "password=$BIGIP_PASSWORD" bigip-creds
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
                      bandwidth:
                        type: string
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
  name: ces-controller
  namespace: $K8S_NAMESPACE
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ces-controller
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
      - "apps"
    resources:
      - deployments
    verbs:
      - get
      - list
      - watch
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
      - clusteregressrules/status
      - namespaceegressrules/status
      - serviceegressrules/status
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
  ces-conf.yaml: |-
    clusterName: k8s
    isSupportRouteDomain: false
    ##AS3 basic configuration
    ##Multi-cluster docking single BIG-IP, controller Common init and remote log
    masterCluster: k8s
    schemaVersion: "3.28.0"
    iRule:
      - bwc-1mbps-irule
      - bwc-2mbps-irule
      - bwc-3mbps-irule
    logPool:
      enableRemoteLog: false
      serverAddresses:
        - "1.2.3.4"
      template: '{
                     "k8s_afm_hsl_log_profile": {
                         "network": {
                             "publisher": {
                                 "use": "/Common/Shared/k8s_firewall_hsl_log_publisher"
                             },
                             "storageFormat": {
                                 "fields": [
                                     "bigip-hostname",
                                     "acl-rule-name",
                                     "acl-policy-name",
                                     "acl-policy-type",
                                     "protocol",
                                     "action",
                                     "drop-reason",
                                     "context-name",
                                     "context-type",
                                     "date-time",
                                     "src-ip",
                                     "src-port",
                                     "vlan",
                                     "route-domain",
                                     "dest-ip",
                                     "dest-port"
                                 ]
                             },
                             "logRuleMatchAccepts": true,
                             "logRuleMatchRejects": true,
                             "logRuleMatchDrops": true,
                             "logIpErrors": true,
                             "logTcpErrors": true,
                             "logTcpEvents": true
                         },
                         "class": "Security_Log_Profile"
                     },
                     "k8s_firewall_hsl_log_publisher": {
                         "destinations": [
                             {
                                 "use": "/Common/Shared/k8s_remote-hsl-dest"
                             },
                             {
                                 "use": "/Common/Shared/k8s_remote-hsl-dest-format"
                             },
                             {
                                 "bigip": "/Common/local-db"
                             }
                         ],
                         "class": "Log_Publisher"
                     },
                     "k8s_remote-hsl-dest": {
                         "pool": {
                             "use": "/Common/Shared/k8s_log_pool"
                         },
                         "class": "Log_Destination",
                         "type": "remote-high-speed-log"
                     },
                     "k8s_remote-hsl-dest-format": {
                         "format": "rfc5424",
                         "remoteHighSpeedLog": {
                             "use": "/Common/Shared/k8s_remote-hsl-dest"
                         },
                         "class": "Log_Destination",
                         "type": "remote-syslog"
                     }
                 }'
    tenant:
      ##common partiton config, init AS3 needs
      - name: "Common"
        namespaces: ""
        virtualService:
          template: ''
        gwPool:
          serverAddresses:
            - "192.168.10.1"
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
          resources:
            requests:
              cpu: '1'
              memory: 1Gi
            limits:
              cpu: '1'
              memory: 1Gi
          command:
            - /ces-controller
            - --bigip-url=$BIGIP_URL
            - --bigip-insecure=$BIGIP_INSECURE
            - --bigip-creds-dir=/ces/bigip-creds
            - --bigip-conf-dir=/ces
          volumeMounts:
            - name: bigip-creds
              mountPath: "/ces/bigip-creds"
              readOnly: true
            - name: bigip-config
              mountPath: /ces
              readOnly: true
      volumes:
        - name: bigip-creds
          secret:
            secretName: bigip-creds
        - name: bigip-config
          configMap:
            name: ces-controller-configmap
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      terminationGracePeriodSeconds: 30
EOF
echo "-------------------------------"
echo ""

echo "[Step 5] Wait CES Controller to Be Ready"
sleep 1s
kubectl -n $K8S_NAMESPACE wait pod --for=condition=Ready -l app=ces-controller
