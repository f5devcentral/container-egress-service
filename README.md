# ces-controller
F5 BigIP AS3 Controller

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
