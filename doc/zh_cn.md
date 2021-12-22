
###log pool template
```
{
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
}    
```

```yaml
##以下是ces-controller-configmap的ces-config.yaml的配置:

clusterName: k8s
isSupportRouteDomain: true
##AS3 basic configuration
##Multi-cluster docking single BIG-IP, controller Common init and remote log
masterCluster: k8s
schemaVersion: "3.29.0"
iRule:
  - bwc-1mbps-irule
  - bwc-2mbps-irule
  - bwc-3mbps-irule
tenant:
  ##common partiton config, init AS3 needs
  - name: "Common"
    namespaces: "default,kube-system"
    virtualService:
      template: ''
      virtualAddresses:
        virtualAddress: "0.0.0.0"
        icmpEcho: "disable"
        arpEnabled: false
        template: ''
    gwPool:
      serverAddresses:
        - "192.168.10.1"
  - name: project2
    namespaces: project2
    routeDomain:
      id: 2
      name: "rd2"
    virtualService:
      template: ""
      virtualAddresses:
        template: '{
              "class": "Service_Address",
              "virtualAddress": "0.0.0.0",
              "icmpEcho": "disable",
              "arpEnabled": false
        }'
    gwPool:
      serverAddresses:
        - "1.16.10.22"
        - "192.168.10.22"
  - name: project3
    namespaces: project3,test-ns-a
    routeDomain:
      id: 4
      name: "rd4"
    virtualService:
      template: '{
                 "layer4": "any",
                 "translateServerAddress": false,
                 "translateServerPort": false,
                 "virtualAddresses": [
                     "1.1.0.0"
                 ],
                 "policyFirewallEnforced": {
                     "use": "/{{tenant}}/Shared/k8s_svc_policy_rd3"
                 },
                 "securityLogProfiles": [
                     {
                         "use": "/{{tenant}}/Shared/k8s_afm_hsl_log_profile"
                     }
                 ],
                 "virtualPort": 0,
                 "snat": "auto",
                 "class": "Service_L4",
                 "pool": "k8s_gw_pool"
             }'
      virtualAddresses:
        virtualAddress: "0.0.0.0"
        icmpEcho: "disable"
        arpEnabled: false
        template: ''     
    gwPool:
      serverAddresses:
        - "10.16.10.23"
logPool:
  loggingEnabled: true
  enableRemoteLog: false
  serverAddresses:
    - "1.2.3.4"
  template: '{
                 "k8s_afm_hsl_log_profile": {
                     "network": {
                         "publisher": {
                             "use": "/{{tenant}}/Shared/k8s_firewall_hsl_log_publisher"
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
                             "use": "/{{tenant}}/Shared/k8s_remote-hsl-dest"
                         },
                         {
                             "use": "/{{tenant}}/Shared/k8s_remote-hsl-dest-format"
                         },
                         {
                             "bigip": "/{{tenant}}/local-db"
                         }
                     ],
                     "class": "Log_Publisher"
                 },
                 "k8s_remote-hsl-dest": {
                     "pool": {
                         "use": "/{{tenant}}/Shared/k8s_log_pool"
                     },
                     "class": "Log_Destination",
                     "type": "remote-high-speed-log"
                 },
                 "k8s_remote-hsl-dest-format": {
                     "format": "rfc5424",
                     "remoteHighSpeedLog": {
                         "use": "/{{tenant}}/Shared/k8s_remote-hsl-dest"
                     },
                     "class": "Log_Destination",
                     "type": "remote-syslog"
                 }
             }'
```

###上面配置参数说明：

```
clusterName：             当前集群名称，用于rule的规则前缀

isSupportRouteDomain：    是否支持严格的RouteDomain

masterCluster：           对于多集群对应单BIG-IP时，需要设置，控制初始化Common tenant

schemaVersion：           AS3中ADC的版本，默认为3.29.0

iRule：                   流量控制配置，此参数需优先在BIG-IP中设置好。

tenant：
   name:                  tenant的名称，对应BIG-IP中的partition
   namespaces：           tenant对应的命名空间，多个可以用逗号隔开，eg: 不支持rd时。此参数可控制监听的namespace下的资源
   virtualService：       ##VS
     template:            VS的模板。用户可自行定义，需要满足AS3规范，具体看上面实例。
     virtualAddresses：   ##virtualAddresses
       virtualAddress:    serviceAddress中virtualAddresses的值。
       icmpEcho:          serviceAddress中icmp的配置
       arpEnabled:        serviceAddress中arp的配置
       template:          serviceAddress的模板设置
   gwPool：               ####gateway
     serverAddresses:     gwpool中的参数值，gateway的ip列表
   logPool：              ##日志
     loggingEnabled：     是否配置log profile
     enableRemoteLog：    是否开启远程日志
     serverAddresses：    pool中的ip列表
     template：           日志配置模板。可参考上面实例
   
```

```

ces-controller-configmap中的参数initialized： 
   "true"服务启动是否要初始化as3,否则为false.
```

##打包：
 
```make release```

##部署：

先修改  install.sh 中的ces-controller-configmap，配置好各参数

```
设置好环境变量：
BIGIP_URL： BIG-IP服务的ip
BIGIP_USERNAME： BIG-IP的用户名
BIGIP_INSECURE： BIG-IP的密码
```

然后执行install.sh 脚本

卸载直接执行uninstall.sh脚本，会删除部署的所有资源。
