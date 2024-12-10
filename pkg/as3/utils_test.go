package as3

import (
	"encoding/json"
	"flag"
	"fmt"
	"k8s.io/klog/v2"
	"testing"

	kubeovnv1alpha1 "github.com/kubeovn/ces-controller/pkg/apis/kubeovn.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetAs3ADC(t *testing.T) {
	aa := map[string]interface{}{
		"tenant1": map[string]interface{}{
			"Shared": map[string]interface{}{
				"aa": "aa",
			},
		},
	}
	bb := as3ADC(aa)
	cc := bb.getAS3SharedApp("tenant1")
	fmt.Println(cc)
}

func TestPatchPolicyIndex(t *testing.T) {
	as3cfg := As3Config{
		ClusterName:          "dwb",
		IsSupportRouteDomain: true,
		Tenant: []TenantConfig{
			{
				Name:       "partition2",
				Namespaces: "test1",
				RouteDomain: RouteDomain{
					Id:   2,
					Name: "rd2",
				},
				VirtualService: VirtualService{},
				Gwpool: Gwpool{
					ServerAddresses: []string{
						"1.1.6.1",
					},
				},
			}, {
				Name:       "partition3",
				Namespaces: "test2",
				RouteDomain: RouteDomain{
					Id:   3,
					Name: "rd3",
				},
				VirtualService: VirtualService{},
				Gwpool: Gwpool{
					ServerAddresses: []string{
						"1.1.6.1",
					},
				},
			},
		},
	}
	initTenantConfig(as3cfg, "kube-system")

	body := PatchBody{}

	aa := FirewallPolicy{
		Rules: []Use{
			//{
			//	Use: "/Common/Shared/dwb_svc_test1_svcgr1_ext_exsvc-svc1_rule_list",
			//},
			//{
			//	Use: "/Common/Shared/dwb_svc_test1_svcgr2_ext_exsvc-svc2_rule_list",
			//},
			//{
			//	Use: "/Common/Shared/dwb_svc_test2_svcgr3_ext_exsvc-svc3_rule_list",
			//},
			//{
			//	Use: "/Common/Shared/dwb_svc_test2_svcgr4_ext_exsvc-svc4_rule_list",
			//},
		},
	}

	bb := FirewallPolicy{
		Rules: []Use{
			{
				Use: "/Common/Shared/dwb_svc_test2_svcgr4_ext_exsvc-svc4_rule_list",
			},
		},
	}
	path := "/Common/Shared/dwb_svc_policy_rd"
	printObj(policyPatchJson(aa, bb, path, body, false))
}

func printObj(obj interface{}) {
	data, _ := json.Marshal(obj)
	fmt.Println(string(data))
}

func TestGetPartiton(t *testing.T) {
	adcStr := `{
    "Common": {
        "Shared": {
            "class": "Application",
            "dwb_global_cgr1_ext_exsvc-c1_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "211.6.6.7"
                ]
            },
            "dwb_global_cgr1_ext_exsvc-c1_ports_tcp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_global_cgr1_ext_exsvc-c1_ports_udp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_global_cgr1_ext_exsvc-c1_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "udp",
                        "name": "accept_exsvc-c1_udp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_global_cgr1_ext_exsvc-c1_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_global_cgr1_ext_exsvc-c1_ports_udp"
                                }
                            ]
                        },
                        "source": {},
                        "action": "accept"
                    },
                    {
                        "protocol": "tcp",
                        "name": "accept_exsvc-c1_tcp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_global_cgr1_ext_exsvc-c1_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_global_cgr1_ext_exsvc-c1_ports_tcp"
                                }
                            ]
                        },
                        "source": {},
                        "action": "accept"
                    }
                ]
            },
            "dwb_global_cgr2_ext_exsvc-c2_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "211.6.6.7"
                ]
            },
            "dwb_global_cgr2_ext_exsvc-c2_ports_tcp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_global_cgr2_ext_exsvc-c2_ports_udp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_global_cgr2_ext_exsvc-c2_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "udp",
                        "name": "accept_exsvc-c2_udp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_global_cgr2_ext_exsvc-c2_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_global_cgr2_ext_exsvc-c2_ports_udp"
                                }
                            ]
                        },
                        "source": {},
                        "action": "accept"
                    },
                    {
                        "protocol": "tcp",
                        "name": "accept_exsvc-c2_tcp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_global_cgr2_ext_exsvc-c2_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_global_cgr2_ext_exsvc-c2_ports_tcp"
                                }
                            ]
                        },
                        "source": {},
                        "action": "accept"
                    }
                ]
            },
            "dwb_global_cgr3_ext_exsvc-c3_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "211.6.6.7"
                ]
            },
            "dwb_global_cgr3_ext_exsvc-c3_ports_tcp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_global_cgr3_ext_exsvc-c3_ports_udp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_global_cgr3_ext_exsvc-c3_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "udp",
                        "name": "accept_exsvc-c3_udp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_global_cgr3_ext_exsvc-c3_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_global_cgr3_ext_exsvc-c3_ports_udp"
                                }
                            ]
                        },
                        "source": {},
                        "action": "accept"
                    },
                    {
                        "protocol": "tcp",
                        "name": "accept_exsvc-c3_tcp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_global_cgr3_ext_exsvc-c3_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_global_cgr3_ext_exsvc-c3_ports_tcp"
                                }
                            ]
                        },
                        "source": {},
                        "action": "accept"
                    }
                ]
            },
            "dwb_global_cgr4_ext_exsvc-c4_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "211.6.6.7"
                ]
            },
            "dwb_global_cgr4_ext_exsvc-c4_ports_tcp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_global_cgr4_ext_exsvc-c4_ports_udp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_global_cgr4_ext_exsvc-c4_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "udp",
                        "name": "accept_exsvc-c4_udp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_global_cgr4_ext_exsvc-c4_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_global_cgr4_ext_exsvc-c4_ports_udp"
                                }
                            ]
                        },
                        "source": {},
                        "action": "accept"
                    },
                    {
                        "protocol": "tcp",
                        "name": "accept_exsvc-c4_tcp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_global_cgr4_ext_exsvc-c4_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_global_cgr4_ext_exsvc-c4_ports_tcp"
                                }
                            ]
                        },
                        "source": {},
                        "action": "accept"
                    }
                ]
            },
            "dwb_gw_pool": {
                "class": "Pool",
                "members": [
                    {
                        "serverAddresses": [
                            "1.1.5.1"
                        ],
                        "enable": true,
                        "servicePort": 0
                    }
                ],
                "monitors": [
                    {
                        "bigip": "/Common/gateway_icmp"
                    }
                ]
            },
            "dwb_ns_policy_rd": {
                "class": "Firewall_Policy",
                "rules": [
                    {
                        "use": "/Common/Shared/dwb_ns_test2_nsgr3_ext_exsvc-ns3_rule_list"
                    },
                    {
                        "use": "/Common/Shared/dwb_ns_test2_nsgr4_ext_exsvc-ns4_rule_list"
                    },
                    {
                        "use": "/Common/Shared/dwb_ns_test1_nsgr1_ext_exsvc-ns1_rule_list"
                    },
                    {
                        "use": "/Common/Shared/dwb_ns_test1_nsgr2_ext_exsvc-ns2_rule_list"
                    }
                ]
            },
            "dwb_ns_test1_nsgr1_ext_exsvc-ns1_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "211.6.6.7"
                ]
            },
            "dwb_ns_test1_nsgr1_ext_exsvc-ns1_ports_tcp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_ns_test1_nsgr1_ext_exsvc-ns1_ports_udp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_ns_test1_nsgr1_ext_exsvc-ns1_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "udp",
                        "name": "accept_exsvc-ns1_udp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test1_nsgr1_ext_exsvc-ns1_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test1_nsgr1_ext_exsvc-ns1_ports_udp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test1_nsgr1_ext_exsvc-ns1_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    },
                    {
                        "protocol": "tcp",
                        "name": "accept_exsvc-ns1_tcp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test1_nsgr1_ext_exsvc-ns1_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test1_nsgr1_ext_exsvc-ns1_ports_tcp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test1_nsgr1_ext_exsvc-ns1_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    }
                ]
            },
            "dwb_ns_test1_nsgr1_ext_exsvc-ns1_src_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "10.3.0.0/16"
                ]
            },
            "dwb_ns_test1_nsgr2_ext_exsvc-ns2_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "211.6.6.7"
                ]
            },
            "dwb_ns_test1_nsgr2_ext_exsvc-ns2_ports_tcp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_ns_test1_nsgr2_ext_exsvc-ns2_ports_udp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_ns_test1_nsgr2_ext_exsvc-ns2_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "udp",
                        "name": "accept_exsvc-ns2_udp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test1_nsgr2_ext_exsvc-ns2_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test1_nsgr2_ext_exsvc-ns2_ports_udp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test1_nsgr2_ext_exsvc-ns2_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    },
                    {
                        "protocol": "tcp",
                        "name": "accept_exsvc-ns2_tcp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test1_nsgr2_ext_exsvc-ns2_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test1_nsgr2_ext_exsvc-ns2_ports_tcp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test1_nsgr2_ext_exsvc-ns2_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    }
                ]
            },
            "dwb_ns_test1_nsgr2_ext_exsvc-ns2_src_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "10.3.0.0/16"
                ]
            },
            "dwb_ns_test2_nsgr3_ext_exsvc-ns3_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "211.6.6.7"
                ]
            },
            "dwb_ns_test2_nsgr3_ext_exsvc-ns3_ports_tcp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_ns_test2_nsgr3_ext_exsvc-ns3_ports_udp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_ns_test2_nsgr3_ext_exsvc-ns3_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "tcp",
                        "name": "accept_exsvc-ns3_tcp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test2_nsgr3_ext_exsvc-ns3_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test2_nsgr3_ext_exsvc-ns3_ports_tcp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test2_nsgr3_ext_exsvc-ns3_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    },
                    {
                        "protocol": "udp",
                        "name": "accept_exsvc-ns3_udp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test2_nsgr3_ext_exsvc-ns3_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test2_nsgr3_ext_exsvc-ns3_ports_udp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test2_nsgr3_ext_exsvc-ns3_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    }
                ]
            },
            "dwb_ns_test2_nsgr3_ext_exsvc-ns3_src_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "10.4.0.0/16"
                ]
            },
            "dwb_ns_test2_nsgr4_ext_exsvc-ns4_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "211.6.6.7"
                ]
            },
            "dwb_ns_test2_nsgr4_ext_exsvc-ns4_ports_tcp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_ns_test2_nsgr4_ext_exsvc-ns4_ports_udp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_ns_test2_nsgr4_ext_exsvc-ns4_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "udp",
                        "name": "accept_exsvc-ns4_udp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test2_nsgr4_ext_exsvc-ns4_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test2_nsgr4_ext_exsvc-ns4_ports_udp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test2_nsgr4_ext_exsvc-ns4_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    },
                    {
                        "protocol": "tcp",
                        "name": "accept_exsvc-ns4_tcp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test2_nsgr4_ext_exsvc-ns4_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test2_nsgr4_ext_exsvc-ns4_ports_tcp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_ns_test2_nsgr4_ext_exsvc-ns4_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    }
                ]
            },
            "dwb_ns_test2_nsgr4_ext_exsvc-ns4_src_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "10.4.0.0/16"
                ]
            },
            "dwb_outbound_vs": {
                "layer4": "any",
                "translateServerAddress": false,
                "translateServerPort": false,
                "virtualAddresses": [
                    "0.0.0.0"
                ],
                "policyFirewallEnforced": {
                    "use": "/Common/Shared/dwb_svc_policy_rd"
                },
                "virtualPort": 0,
                "snat": "auto",
                "class": "Service_L4",
                "pool": "dwb_gw_pool"
            },
            "dwb_svc_deny_all_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "any",
                        "name": "deny_all_rule",
                        "destination": {},
                        "source": {},
                        "action": "drop"
                    }
                ]
            },
            "dwb_svc_policy_rd": {
                "class": "Firewall_Policy",
                "rules": [
                    {
                        "use": "/Common/Shared/dwb_svc_deny_all_rule_list"
                    },
                    {
                        "use": "/Common/Shared/dwb_svc_test1_svcgr1_ext_exsvc-svc1_rule_list"
                    },
                    {
                        "use": "/Common/Shared/dwb_svc_test1_svcgr2_ext_exsvc-svc2_rule_list"
                    },
                    {
                        "use": "/Common/Shared/dwb_svc_test2_svcgr3_ext_exsvc-svc3_rule_list"
                    },
                    {
                        "use": "/Common/Shared/dwb_svc_test2_svcgr4_ext_exsvc-svc4_rule_list"
                    }
                ]
            },
            "dwb_svc_test1_svcgr1_ext_exsvc-svc1_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "211.6.6.7"
                ]
            },
            "dwb_svc_test1_svcgr1_ext_exsvc-svc1_ports_tcp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_svc_test1_svcgr1_ext_exsvc-svc1_ports_udp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_svc_test1_svcgr1_ext_exsvc-svc1_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "udp",
                        "name": "accept_exsvc-svc1_udp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test1_svcgr1_ext_exsvc-svc1_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test1_svcgr1_ext_exsvc-svc1_ports_udp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test1_svcgr1_ext_exsvc-svc1_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    },
                    {
                        "protocol": "tcp",
                        "name": "accept_exsvc-svc1_tcp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test1_svcgr1_ext_exsvc-svc1_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test1_svcgr1_ext_exsvc-svc1_ports_tcp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test1_svcgr1_ext_exsvc-svc1_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    }
                ]
            },
            "dwb_svc_test1_svcgr1_ext_exsvc-svc1_src_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "10.3.67.23"
                ]
            },
            "dwb_svc_test1_svcgr2_ext_exsvc-svc2_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "211.6.6.7"
                ]
            },
            "dwb_svc_test1_svcgr2_ext_exsvc-svc2_ports_tcp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_svc_test1_svcgr2_ext_exsvc-svc2_ports_udp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_svc_test1_svcgr2_ext_exsvc-svc2_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "udp",
                        "name": "accept_exsvc-svc2_udp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test1_svcgr2_ext_exsvc-svc2_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test1_svcgr2_ext_exsvc-svc2_ports_udp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test1_svcgr2_ext_exsvc-svc2_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    },
                    {
                        "protocol": "tcp",
                        "name": "accept_exsvc-svc2_tcp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test1_svcgr2_ext_exsvc-svc2_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test1_svcgr2_ext_exsvc-svc2_ports_tcp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test1_svcgr2_ext_exsvc-svc2_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    }
                ]
            },
            "dwb_svc_test1_svcgr2_ext_exsvc-svc2_src_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "10.3.67.23"
                ]
            },
            "dwb_svc_test2_svcgr3_ext_exsvc-svc3_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "211.6.6.7"
                ]
            },
            "dwb_svc_test2_svcgr3_ext_exsvc-svc3_ports_tcp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_svc_test2_svcgr3_ext_exsvc-svc3_ports_udp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_svc_test2_svcgr3_ext_exsvc-svc3_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "udp",
                        "name": "accept_exsvc-svc3_udp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test2_svcgr3_ext_exsvc-svc3_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test2_svcgr3_ext_exsvc-svc3_ports_udp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test2_svcgr3_ext_exsvc-svc3_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    },
                    {
                        "protocol": "tcp",
                        "name": "accept_exsvc-svc3_tcp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test2_svcgr3_ext_exsvc-svc3_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test2_svcgr3_ext_exsvc-svc3_ports_tcp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test2_svcgr3_ext_exsvc-svc3_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    }
                ]
            },
            "dwb_svc_test2_svcgr3_ext_exsvc-svc3_src_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "10.3.67.23"
                ]
            },
            "dwb_svc_test2_svcgr4_ext_exsvc-svc4_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "211.6.6.7"
                ]
            },
            "dwb_svc_test2_svcgr4_ext_exsvc-svc4_ports_tcp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_svc_test2_svcgr4_ext_exsvc-svc4_ports_udp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_svc_test2_svcgr4_ext_exsvc-svc4_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "udp",
                        "name": "accept_exsvc-svc4_udp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test2_svcgr4_ext_exsvc-svc4_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test2_svcgr4_ext_exsvc-svc4_ports_udp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test2_svcgr4_ext_exsvc-svc4_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    },
                    {
                        "protocol": "tcp",
                        "name": "accept_exsvc-svc4_tcp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test2_svcgr4_ext_exsvc-svc4_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test2_svcgr4_ext_exsvc-svc4_ports_tcp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/Common/Shared/dwb_svc_test2_svcgr4_ext_exsvc-svc4_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    }
                ]
            },
            "dwb_svc_test2_svcgr4_ext_exsvc-svc4_src_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "10.3.67.23"
                ]
            },
            "dwb_system_global_policy": {
                "class": "Firewall_Policy",
                "rules": [
                    {
                        "use": "/Common/Shared/dwb_global_cgr1_ext_exsvc-c1_rule_list"
                    },
                    {
                        "use": "/Common/Shared/dwb_global_cgr2_ext_exsvc-c2_rule_list"
                    },
                    {
                        "use": "/Common/Shared/dwb_global_cgr3_ext_exsvc-c3_rule_list"
                    },
                    {
                        "use": "/Common/Shared/dwb_global_cgr4_ext_exsvc-c4_rule_list"
                    }
                ]
            },
            "template": "shared"
        },
        "class": "Tenant"
    },
    "partition2": {
        "Shared": {
            "class": "Application",
            "dwb_gw_pool": {
                "class": "Pool",
                "members": [
                    {
                        "serverAddresses": [
                            "1.1.6.1"
                        ],
                        "enable": true,
                        "servicePort": 0
                    }
                ],
                "monitors": [
                    {
                        "bigip": "/Common/gateway_icmp"
                    }
                ]
            },
            "dwb_ns_policy_rd2": {
                "class": "Firewall_Policy",
                "rules": [
                    {
                        "use": "/partition2/Shared/dwb_ns_test1_nsgr1_ext_exsvc-ns1_rule_list"
                    },
                    {
                        "use": "/partition2/Shared/dwb_ns_test1_nsgr2_ext_exsvc-ns2_rule_list"
                    }
                ]
            },
            "dwb_ns_test1_nsgr1_ext_exsvc-ns1_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "211.6.6.7"
                ]
            },
            "dwb_ns_test1_nsgr1_ext_exsvc-ns1_ports_tcp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_ns_test1_nsgr1_ext_exsvc-ns1_ports_udp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_ns_test1_nsgr1_ext_exsvc-ns1_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "udp",
                        "name": "accept_exsvc-ns1_udp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/partition2/Shared/dwb_ns_test1_nsgr1_ext_exsvc-ns1_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/partition2/Shared/dwb_ns_test1_nsgr1_ext_exsvc-ns1_ports_udp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/partition2/Shared/dwb_ns_test1_nsgr1_ext_exsvc-ns1_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    },
                    {
                        "protocol": "tcp",
                        "name": "accept_exsvc-ns1_tcp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/partition2/Shared/dwb_ns_test1_nsgr1_ext_exsvc-ns1_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/partition2/Shared/dwb_ns_test1_nsgr1_ext_exsvc-ns1_ports_tcp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/partition2/Shared/dwb_ns_test1_nsgr1_ext_exsvc-ns1_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    }
                ]
            },
            "dwb_ns_test1_nsgr1_ext_exsvc-ns1_src_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "10.3.0.0/16"
                ]
            },
            "dwb_ns_test1_nsgr2_ext_exsvc-ns2_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "211.6.6.7"
                ]
            },
            "dwb_ns_test1_nsgr2_ext_exsvc-ns2_ports_tcp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_ns_test1_nsgr2_ext_exsvc-ns2_ports_udp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_ns_test1_nsgr2_ext_exsvc-ns2_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "tcp",
                        "name": "accept_exsvc-ns2_tcp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/partition2/Shared/dwb_ns_test1_nsgr2_ext_exsvc-ns2_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/partition2/Shared/dwb_ns_test1_nsgr2_ext_exsvc-ns2_ports_tcp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/partition2/Shared/dwb_ns_test1_nsgr2_ext_exsvc-ns2_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    },
                    {
                        "protocol": "udp",
                        "name": "accept_exsvc-ns2_udp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/partition2/Shared/dwb_ns_test1_nsgr2_ext_exsvc-ns2_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/partition2/Shared/dwb_ns_test1_nsgr2_ext_exsvc-ns2_ports_udp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/partition2/Shared/dwb_ns_test1_nsgr2_ext_exsvc-ns2_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    }
                ]
            },
            "dwb_ns_test1_nsgr2_ext_exsvc-ns2_src_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "10.3.0.0/16"
                ]
            },
            "dwb_outbound_vs": {
                "layer4": "any",
                "translateServerAddress": false,
                "translateServerPort": false,
                "virtualAddresses": [
                    "0.0.0.0"
                ],
                "policyFirewallEnforced": {
                    "use": "/partition2/Shared/dwb_svc_policy_rd2"
                },
                "virtualPort": 0,
                "snat": "auto",
                "class": "Service_L4",
                "pool": "dwb_gw_pool"
            },
            "dwb_svc_deny_all_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "any",
                        "name": "deny_all_rule",
                        "destination": {},
                        "source": {},
                        "action": "drop"
                    }
                ]
            },
            "dwb_svc_policy_rd2": {
                "class": "Firewall_Policy",
                "rules": [
                    {
                        "use": "/partition2/Shared/dwb_svc_deny_all_rule_list"
                    },
                    {
                        "use": "/partition2/Shared/dwb_svc_test1_svcgr1_ext_exsvc-svc1_rule_list"
                    },
                    {
                        "use": "/partition2/Shared/dwb_svc_test1_svcgr2_ext_exsvc-svc2_rule_list"
                    }
                ]
            },
            "dwb_svc_test1_svcgr1_ext_exsvc-svc1_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "211.6.6.7"
                ]
            },
            "dwb_svc_test1_svcgr1_ext_exsvc-svc1_ports_tcp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_svc_test1_svcgr1_ext_exsvc-svc1_ports_udp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_svc_test1_svcgr1_ext_exsvc-svc1_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "udp",
                        "name": "accept_exsvc-svc1_udp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/partition2/Shared/dwb_svc_test1_svcgr1_ext_exsvc-svc1_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/partition2/Shared/dwb_svc_test1_svcgr1_ext_exsvc-svc1_ports_udp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/partition2/Shared/dwb_svc_test1_svcgr1_ext_exsvc-svc1_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    },
                    {
                        "protocol": "tcp",
                        "name": "accept_exsvc-svc1_tcp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/partition2/Shared/dwb_svc_test1_svcgr1_ext_exsvc-svc1_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/partition2/Shared/dwb_svc_test1_svcgr1_ext_exsvc-svc1_ports_tcp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/partition2/Shared/dwb_svc_test1_svcgr1_ext_exsvc-svc1_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    }
                ]
            },
            "dwb_svc_test1_svcgr1_ext_exsvc-svc1_src_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "10.3.67.23"
                ]
            },
            "dwb_svc_test1_svcgr2_ext_exsvc-svc2_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "211.6.6.7"
                ]
            },
            "dwb_svc_test1_svcgr2_ext_exsvc-svc2_ports_tcp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_svc_test1_svcgr2_ext_exsvc-svc2_ports_udp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_svc_test1_svcgr2_ext_exsvc-svc2_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "udp",
                        "name": "accept_exsvc-svc2_udp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/partition2/Shared/dwb_svc_test1_svcgr2_ext_exsvc-svc2_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/partition2/Shared/dwb_svc_test1_svcgr2_ext_exsvc-svc2_ports_udp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/partition2/Shared/dwb_svc_test1_svcgr2_ext_exsvc-svc2_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    },
                    {
                        "protocol": "tcp",
                        "name": "accept_exsvc-svc2_tcp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/partition2/Shared/dwb_svc_test1_svcgr2_ext_exsvc-svc2_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/partition2/Shared/dwb_svc_test1_svcgr2_ext_exsvc-svc2_ports_tcp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/partition2/Shared/dwb_svc_test1_svcgr2_ext_exsvc-svc2_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    }
                ]
            },
            "dwb_svc_test1_svcgr2_ext_exsvc-svc2_src_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "10.3.67.23"
                ]
            },
            "template": "shared"
        },
        "class": "Tenant",
        "defaultRouteDomain": 2
    },
    "partition3": {
        "Shared": {
            "class": "Application",
            "dwb_gw_pool": {
                "class": "Pool",
                "members": [
                    {
                        "serverAddresses": [
                            "1.1.6.1"
                        ],
                        "enable": true,
                        "servicePort": 0
                    }
                ],
                "monitors": [
                    {
                        "bigip": "/Common/gateway_icmp"
                    }
                ]
            },
            "dwb_ns_policy_rd3": {
                "class": "Firewall_Policy",
                "rules": [
                    {
                        "use": "/partition3/Shared/dwb_ns_test2_nsgr3_ext_exsvc-ns3_rule_list"
                    },
                    {
                        "use": "/partition3/Shared/dwb_ns_test2_nsgr4_ext_exsvc-ns4_rule_list"
                    }
                ]
            },
            "dwb_ns_test2_nsgr3_ext_exsvc-ns3_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "211.6.6.7"
                ]
            },
            "dwb_ns_test2_nsgr3_ext_exsvc-ns3_ports_tcp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_ns_test2_nsgr3_ext_exsvc-ns3_ports_udp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_ns_test2_nsgr3_ext_exsvc-ns3_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "udp",
                        "name": "accept_exsvc-ns3_udp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/partition3/Shared/dwb_ns_test2_nsgr3_ext_exsvc-ns3_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/partition3/Shared/dwb_ns_test2_nsgr3_ext_exsvc-ns3_ports_udp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/partition3/Shared/dwb_ns_test2_nsgr3_ext_exsvc-ns3_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    },
                    {
                        "protocol": "tcp",
                        "name": "accept_exsvc-ns3_tcp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/partition3/Shared/dwb_ns_test2_nsgr3_ext_exsvc-ns3_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/partition3/Shared/dwb_ns_test2_nsgr3_ext_exsvc-ns3_ports_tcp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/partition3/Shared/dwb_ns_test2_nsgr3_ext_exsvc-ns3_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    }
                ]
            },
            "dwb_ns_test2_nsgr3_ext_exsvc-ns3_src_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "10.4.0.0/16"
                ]
            },
            "dwb_ns_test2_nsgr4_ext_exsvc-ns4_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "211.6.6.7"
                ]
            },
            "dwb_ns_test2_nsgr4_ext_exsvc-ns4_ports_tcp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_ns_test2_nsgr4_ext_exsvc-ns4_ports_udp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_ns_test2_nsgr4_ext_exsvc-ns4_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "udp",
                        "name": "accept_exsvc-ns4_udp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/partition3/Shared/dwb_ns_test2_nsgr4_ext_exsvc-ns4_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/partition3/Shared/dwb_ns_test2_nsgr4_ext_exsvc-ns4_ports_udp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/partition3/Shared/dwb_ns_test2_nsgr4_ext_exsvc-ns4_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    },
                    {
                        "protocol": "tcp",
                        "name": "accept_exsvc-ns4_tcp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/partition3/Shared/dwb_ns_test2_nsgr4_ext_exsvc-ns4_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/partition3/Shared/dwb_ns_test2_nsgr4_ext_exsvc-ns4_ports_tcp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/partition3/Shared/dwb_ns_test2_nsgr4_ext_exsvc-ns4_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    }
                ]
            },
            "dwb_ns_test2_nsgr4_ext_exsvc-ns4_src_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "10.4.0.0/16"
                ]
            },
            "dwb_outbound_vs": {
                "layer4": "any",
                "translateServerAddress": false,
                "translateServerPort": false,
                "virtualAddresses": [
                    "0.0.0.0"
                ],
                "policyFirewallEnforced": {
                    "use": "/partition3/Shared/dwb_svc_policy_rd3"
                },
                "virtualPort": 0,
                "snat": "auto",
                "class": "Service_L4",
                "pool": "dwb_gw_pool"
            },
            "dwb_svc_deny_all_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "any",
                        "name": "deny_all_rule",
                        "destination": {},
                        "source": {},
                        "action": "drop"
                    }
                ]
            },
            "dwb_svc_policy_rd3": {
                "class": "Firewall_Policy",
                "rules": [
                    {
                        "use": "/partition3/Shared/dwb_svc_deny_all_rule_list"
                    },
                    {
                        "use": "/partition3/Shared/dwb_svc_test2_svcgr3_ext_exsvc-svc3_rule_list"
                    },
                    {
                        "use": "/partition3/Shared/dwb_svc_test2_svcgr4_ext_exsvc-svc4_rule_list"
                    }
                ]
            },
            "dwb_svc_test2_svcgr3_ext_exsvc-svc3_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "211.6.6.7"
                ]
            },
            "dwb_svc_test2_svcgr3_ext_exsvc-svc3_ports_tcp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_svc_test2_svcgr3_ext_exsvc-svc3_ports_udp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_svc_test2_svcgr3_ext_exsvc-svc3_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "udp",
                        "name": "accept_exsvc-svc3_udp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/partition3/Shared/dwb_svc_test2_svcgr3_ext_exsvc-svc3_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/partition3/Shared/dwb_svc_test2_svcgr3_ext_exsvc-svc3_ports_udp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/partition3/Shared/dwb_svc_test2_svcgr3_ext_exsvc-svc3_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    },
                    {
                        "protocol": "tcp",
                        "name": "accept_exsvc-svc3_tcp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/partition3/Shared/dwb_svc_test2_svcgr3_ext_exsvc-svc3_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/partition3/Shared/dwb_svc_test2_svcgr3_ext_exsvc-svc3_ports_tcp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/partition3/Shared/dwb_svc_test2_svcgr3_ext_exsvc-svc3_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    }
                ]
            },
            "dwb_svc_test2_svcgr3_ext_exsvc-svc3_src_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "10.3.67.23"
                ]
            },
            "dwb_svc_test2_svcgr4_ext_exsvc-svc4_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "211.6.6.7"
                ]
            },
            "dwb_svc_test2_svcgr4_ext_exsvc-svc4_ports_tcp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_svc_test2_svcgr4_ext_exsvc-svc4_ports_udp": {
                "class": "Firewall_Port_List",
                "ports": [
                    "9090"
                ]
            },
            "dwb_svc_test2_svcgr4_ext_exsvc-svc4_rule_list": {
                "class": "Firewall_Rule_List",
                "rules": [
                    {
                        "protocol": "udp",
                        "name": "accept_exsvc-svc4_udp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/partition3/Shared/dwb_svc_test2_svcgr4_ext_exsvc-svc4_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/partition3/Shared/dwb_svc_test2_svcgr4_ext_exsvc-svc4_ports_udp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/partition3/Shared/dwb_svc_test2_svcgr4_ext_exsvc-svc4_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    },
                    {
                        "protocol": "tcp",
                        "name": "accept_exsvc-svc4_tcp",
                        "destination": {
                            "addressLists": [
                                {
                                    "use": "/partition3/Shared/dwb_svc_test2_svcgr4_ext_exsvc-svc4_address"
                                }
                            ],
                            "portLists": [
                                {
                                    "use": "/partition3/Shared/dwb_svc_test2_svcgr4_ext_exsvc-svc4_ports_tcp"
                                }
                            ]
                        },
                        "source": {
                            "addressLists": [
                                {
                                    "use": "/partition3/Shared/dwb_svc_test2_svcgr4_ext_exsvc-svc4_src_address"
                                }
                            ]
                        },
                        "action": "accept"
                    }
                ]
            },
            "dwb_svc_test2_svcgr4_ext_exsvc-svc4_src_address": {
                "class": "Firewall_Address_List",
                "addresses": [
                    "10.3.67.23"
                ]
            },
            "template": "shared"
        },
        "class": "Tenant",
        "defaultRouteDomain": 3
    },
    "class": "ADC",
    "id": "k8s-ces-controller",
    "schemaVersion": "3.29.0",
    "updateMode": "selective",
    "controls": {
        "archiveTimestamp": "2021-09-07T09:22:14.510Z"
    }
}
`
	var adc = as3ADC{}
	json.Unmarshal([]byte(adcStr), &adc)
	partition := "partition2"
	printObj(adc.getAS3SharedApp(partition))
}

func TestAs3PostDefaultPartition(t *testing.T) {
	as3cfg := As3Config{
		ClusterName:          "k8s",
		IsSupportRouteDomain: false,
		Tenant: []TenantConfig{
			{
				Name:        DefaultPartition,
				RouteDomain: RouteDomain{},
			},
		},
	}
	initTenantConfig(as3cfg, "kube-system")
	as3 := initDefaultAS3()
	printObj(as3)
}

func TestMockClusteEgressRule(t *testing.T) {
	clusterEgressList := kubeovnv1alpha1.ClusterEgressRuleList{
		Items: []kubeovnv1alpha1.ClusterEgressRule{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: kubeovnv1alpha1.ClusterEgressRuleSpec{
					Action:           "accept",
					ExternalServices: []string{"exsvc-test1", "exsvc-test2"},
				},
			},
		},
	}

	externalServiceList := kubeovnv1alpha1.ExternalServiceList{
		Items: []kubeovnv1alpha1.ExternalService{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exsvc-test1",
					Namespace: "kube-system",
				},
				Spec: kubeovnv1alpha1.ExternalServiceSpec{
					Addresses: []string{
						"192.168.1.0",
						"192.168.2.0",
					},
					Ports: []kubeovnv1alpha1.ExternalServicePort{
						{
							Name:     "tcp1",
							Protocol: "tcp",
							Port:     "8080",
						},
					},
				},
			}, {
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exsvc-test2",
					Namespace: "kube-system",
				},
				Spec: kubeovnv1alpha1.ExternalServiceSpec{
					Addresses: []string{
						"192.168.10.0",
						"192.168.20.0",
					},
					Ports: []kubeovnv1alpha1.ExternalServicePort{
						{
							Name:     "tcp2",
							Protocol: "tcp",
							Port:     "8081",
						}, {
							Name:     "udp2",
							Protocol: "udp",
							Port:     "9090",
						},
					},
				},
			},
		},
	}

	as3cfg := As3Config{
		ClusterName:          "k8s",
		IsSupportRouteDomain: false,
		Tenant: []TenantConfig{
			{
				Name: "Common",
				RouteDomain: RouteDomain{
					Name: "0",
					Id:   0,
				},
				VirtualService: VirtualService{},
				Gwpool: Gwpool{
					ServerAddresses: []string{"192.168.132.1"},
				},
			},
		},
	}
	initTenantConfig(as3cfg, "kube-system")
	tntcfg := GetTenantConfigForParttition(DefaultPartition)

	as3post := newAs3Post(nil, nil, &clusterEgressList, &externalServiceList, nil, nil, nil, tntcfg)
	as3 := initDefaultAS3()
	printObj(as3)
	srcAdc := as3[DeclarationKey].(as3ADC)
	adc := as3ADC{}
	as3post.generateAS3ResourceDeclaration(adc)
	body := fullResource(DefaultPartition, false, srcAdc, adc)
	printObj(body)

	//add the same clusteregressrule at above as3
	clusterEgressList = kubeovnv1alpha1.ClusterEgressRuleList{
		Items: []kubeovnv1alpha1.ClusterEgressRule{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: kubeovnv1alpha1.ClusterEgressRuleSpec{
					Action:           "accept",
					ExternalServices: []string{"exsvc-test1", "exsvc-test2"},
				},
			},
		},
	}

	externalServiceList = kubeovnv1alpha1.ExternalServiceList{
		Items: []kubeovnv1alpha1.ExternalService{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exsvc-test1",
					Namespace: "kube-system",
				},
				Spec: kubeovnv1alpha1.ExternalServiceSpec{
					Addresses: []string{
						"192.168.1.0",
						"192.168.2.0",
					},
					Ports: []kubeovnv1alpha1.ExternalServicePort{
						{
							Name:     "tcp1",
							Protocol: "tcp",
							Port:     "8080",
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exsvc-test2",
					Namespace: "kube-system",
				},
				Spec: kubeovnv1alpha1.ExternalServiceSpec{
					Addresses: []string{
						"192.168.10.0",
						"192.168.20.0",
					},
					Ports: []kubeovnv1alpha1.ExternalServicePort{
						{
							Name:     "tcp2",
							Protocol: "tcp",
							Port:     "8081",
						}, {
							Name:     "udp2",
							Protocol: "udp",
							Port:     "9090",
						},
					},
				},
			},
		},
	}
	as3post = newAs3Post(nil, nil, &clusterEgressList, &externalServiceList, nil, nil, nil, tntcfg)
	deltaAdc := as3ADC{}
	as3post.generateAS3ResourceDeclaration(deltaAdc)

	var srcAs3 map[string]interface{}
	err := validateJSONAndFetchObject(body, &srcAs3)
	if err != nil {
		t.Error(err)
	}
	srcAdc = as3ADC(srcAs3[DeclarationKey].(map[string]interface{}))
	body = fullResource(DefaultPartition, false, srcAdc, deltaAdc)

	printObj(body)

	//add diff clusteregressrule
	clusterEgressList = kubeovnv1alpha1.ClusterEgressRuleList{
		Items: []kubeovnv1alpha1.ClusterEgressRule{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test3",
				},
				Spec: kubeovnv1alpha1.ClusterEgressRuleSpec{
					Action:           "accept",
					ExternalServices: []string{"exsvc-test3"},
				},
			},
		},
	}

	externalServiceList = kubeovnv1alpha1.ExternalServiceList{
		Items: []kubeovnv1alpha1.ExternalService{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exsvc-test3",
					Namespace: "kube-system",
				},
				Spec: kubeovnv1alpha1.ExternalServiceSpec{
					Addresses: []string{
						"182.16.16.19",
						"182.181.1.1",
					},
					Ports: []kubeovnv1alpha1.ExternalServicePort{
						{
							Name:     "tcp1",
							Protocol: "tcp",
							Port:     "1010",
						}, {
							Name:     "upd1",
							Protocol: "udp",
							Port:     "2020",
						},
					},
				},
			},
		},
	}
	as3post = newAs3Post(nil, nil, &clusterEgressList, &externalServiceList, nil, nil, nil, tntcfg)
	deltaAdc = as3ADC{}
	srcAs3 = map[string]interface{}{}
	as3post.generateAS3ResourceDeclaration(deltaAdc)
	err = validateJSONAndFetchObject(body, &srcAs3)
	if err != nil {
		t.Error(err)
	}
	srcAdc = as3ADC(srcAs3[DeclarationKey].(map[string]interface{}))
	body = fullResource(DefaultPartition, false, srcAdc, deltaAdc)
	printObj(body)

	//delete clusteregressrule
	srcAs3 = map[string]interface{}{}
	err = validateJSONAndFetchObject(body, &srcAs3)
	if err != nil {
		t.Error(err)
	}
	srcAdc = as3ADC(srcAs3[DeclarationKey].(map[string]interface{}))
	body = fullResource(DefaultPartition, true, srcAdc, deltaAdc)
	printObj(body)

	//delete only one clusteregressrule
	as3post = newAs3Post(nil, nil, &clusterEgressList, &externalServiceList, nil, nil, nil, tntcfg)
	srcAdc = as3ADC{}
	deltaAdc = as3ADC{}
	as3post.generateAS3ResourceDeclaration(srcAdc)
	printObj(srcAdc)
	as3post.generateAS3ResourceDeclaration(deltaAdc)
	body = fullResource(DefaultPartition, true, srcAdc, deltaAdc)
	printObj(body)

}

func TestMockNamespaceEgressRule(t *testing.T) {
	namespaceEgressRuleList := kubeovnv1alpha1.NamespaceEgressRuleList{
		Items: []kubeovnv1alpha1.NamespaceEgressRule{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "project1",
				},
				Spec: kubeovnv1alpha1.NamespaceEgressRuleSpec{
					Action:           "accept",
					ExternalServices: []string{"exsvc-test1", "exsvc-test2"},
				},
			},
		},
	}
	externalServiceList := kubeovnv1alpha1.ExternalServiceList{
		Items: []kubeovnv1alpha1.ExternalService{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exsvc-test1",
					Namespace: "project1",
				},
				Spec: kubeovnv1alpha1.ExternalServiceSpec{
					Addresses: []string{
						"192.168.1.0",
						"192.168.2.0",
					},
					Ports: []kubeovnv1alpha1.ExternalServicePort{
						{
							Name:     "tcp1",
							Protocol: "tcp",
							Port:     "8080",
						},
					},
				},
			}, {
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exsvc-test2",
					Namespace: "project1",
				},
				Spec: kubeovnv1alpha1.ExternalServiceSpec{
					Addresses: []string{
						"192.168.10.0",
						"192.168.20.0",
					},
					Ports: []kubeovnv1alpha1.ExternalServicePort{
						{
							Name:     "tcp2",
							Protocol: "tcp",
							Port:     "8081",
						}, {
							Name:     "udp2",
							Protocol: "udp",
							Port:     "9090",
						},
					},
				},
			},
		},
	}
	as3cfg := As3Config{
		ClusterName:          "k8s",
		IsSupportRouteDomain: false,
		Tenant: []TenantConfig{
			{
				Name:       "Common",
				Namespaces: "xxxxxxxx",
				RouteDomain: RouteDomain{
					Name: "0",
					Id:   0,
				},
				VirtualService: VirtualService{},
				Gwpool: Gwpool{
					ServerAddresses: []string{"192.168.132.1"},
				},
			},
			{
				Name:       "project1",
				Namespaces: "project1",
				RouteDomain: RouteDomain{
					Name: "rd1",
					Id:   1,
				},
				VirtualService: VirtualService{},
				Gwpool: Gwpool{
					ServerAddresses: []string{"192.101.100.1"},
				},
			}, {
				Name:       "project2",
				Namespaces: "project2",
				RouteDomain: RouteDomain{
					Name: "rd2",
					Id:   2,
				},
				VirtualService: VirtualService{},
				Gwpool: Gwpool{
					ServerAddresses: []string{"192.101.111.1"},
				},
			},
		},
	}

	namespaceList := corev1.NamespaceList{
		Items: []corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "project1",
					Annotations: map[string]string{
						NamespaceCidr: "10.1.0.1/16",
					},
				},
			},
		},
	}
	initTenantConfig(as3cfg, "kube-system")
	tntcfg := GetTenantConfigForParttition(DefaultPartition)

	as3post := newAs3Post(nil, &namespaceEgressRuleList, nil, &externalServiceList, nil, nil, &namespaceList, tntcfg)

	as3 := initDefaultAS3()
	printObj(as3)

	srcAs3 := map[string]interface{}{}
	err := validateJSONAndFetchObject(as3, &srcAs3)
	if err != nil {
		t.Error(err)
	}
	srcAdc := as3ADC(srcAs3[DeclarationKey].(map[string]interface{}))
	deltaAdc := as3ADC{}
	as3post.generateAS3ResourceDeclaration(deltaAdc)
	body := fullResource("Common", false, srcAdc, deltaAdc)
	t.Log("==================>init namespaceegressrule")
	printObj(body)

	//add same namespaceegressrule
	t.Log("==================>add same namespaceegressrule")
	srcAs3 = map[string]interface{}{}
	err = validateJSONAndFetchObject(body, &srcAs3)
	if err != nil {
		t.Error(err)
	}
	srcAdc = as3ADC(srcAs3[DeclarationKey].(map[string]interface{}))
	body = fullResource("Common", false, srcAdc, deltaAdc)
	printObj(body)

	//surpport rd in common
	t.Log("==================>surpport rd")
	as3cfg.IsSupportRouteDomain = true
	initTenantConfig(as3cfg, "kube-system")
	tntcfg = GetTenantConfigForParttition("project1")
	as3post = newAs3Post(nil, &namespaceEgressRuleList, nil, &externalServiceList, nil, nil, &namespaceList, tntcfg)
	deltaAdc = as3ADC{}
	as3post.generateAS3ResourceDeclaration(deltaAdc)
	body = fullResource("project1", false, srcAdc, deltaAdc)
	printObj(body)

	//surpport rd in common
	t.Log("==================>delete partition")
	body = fullResource("project1", true, srcAdc, deltaAdc)
	printObj(body)

	//add diff namespaceegressrule
	t.Log("==================>add diff namespaceegressrule")
	namespaceEgressRuleList = kubeovnv1alpha1.NamespaceEgressRuleList{
		Items: []kubeovnv1alpha1.NamespaceEgressRule{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test2",
					Namespace: "project2",
				},
				Spec: kubeovnv1alpha1.NamespaceEgressRuleSpec{
					Action:           "accept",
					ExternalServices: []string{"exsvc-test3", "exsvc-test4"},
				},
			},
		},
	}
	externalServiceList = kubeovnv1alpha1.ExternalServiceList{
		Items: []kubeovnv1alpha1.ExternalService{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exsvc-test3",
					Namespace: "project2",
				},
				Spec: kubeovnv1alpha1.ExternalServiceSpec{
					Addresses: []string{
						"192.168.3.0",
						"192.168.4.0",
					},
					Ports: []kubeovnv1alpha1.ExternalServicePort{
						{
							Name:     "tcp1",
							Protocol: "tcp",
							Port:     "1010",
						},
					},
				},
			}, {
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exsvc-test4",
					Namespace: "project2",
				},
				Spec: kubeovnv1alpha1.ExternalServiceSpec{
					Addresses: []string{
						"192.168.30.0",
						"192.168.40.0",
					},
					Ports: []kubeovnv1alpha1.ExternalServicePort{
						{
							Name:     "tcp2",
							Protocol: "tcp",
							Port:     "8081",
						}, {
							Name:     "udp2",
							Protocol: "udp",
							Port:     "9090",
						},
					},
				},
			},
		},
	}
	namespaceList = corev1.NamespaceList{
		Items: []corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "project2",
					Annotations: map[string]string{
						NamespaceCidr: "10.1.0.1/16",
					},
				},
			},
		},
	}
	tntcfg = GetTenantConfigForParttition(DefaultPartition)
	as3post1 := newAs3Post(nil, &namespaceEgressRuleList, nil, &externalServiceList, nil, nil, &namespaceList, tntcfg)
	deltaAdc = as3ADC{}
	as3post1.generateAS3ResourceDeclaration(deltaAdc)
	body = fullResource("Common", false, srcAdc, deltaAdc)
	printObj(body)
	//delete namespaceegressrule
	body = fullResource("Common", true, srcAdc, deltaAdc)
	printObj(body)
}

func TestMockServiceEgressRule(t *testing.T) {
	svcgrList := kubeovnv1alpha1.ServiceEgressRuleList{
		Items: []kubeovnv1alpha1.ServiceEgressRule{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svcgr",
					Namespace: "dwb-test",
				},
				Spec: kubeovnv1alpha1.ServiceEgressRuleSpec{
					Action:           "accept",
					Logging:          true,
					Service:          "test",
					ExternalServices: []string{"exsvc"},
				},
			},
		},
	}
	exsvcList := kubeovnv1alpha1.ExternalServiceList{
		Items: []kubeovnv1alpha1.ExternalService{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exsvc",
					Namespace: "dwb-test",
				},
				Spec: kubeovnv1alpha1.ExternalServiceSpec{
					Addresses: []string{
						"192.168.2.2",
					},
					Ports: []kubeovnv1alpha1.ExternalServicePort{
						{
							Name:      "udp-9090",
							Protocol:  "udp",
							Port:      "9090",
							Bandwidth: "bwc-2mbps-irule",
						},
					},
				},
			},
		},
	}

	as3cfg := As3Config{
		ClusterName:          "cck8s",
		IsSupportRouteDomain: false,
		Tenant: []TenantConfig{
			{
				Name:       "Common",
				Namespaces: "dwb-test",
				RouteDomain: RouteDomain{
					Name: "0",
					Id:   0,
				},
				VirtualService: VirtualService{},
				Gwpool: Gwpool{
					ServerAddresses: []string{"192.168.132.1"},
				},
			},
		},
	}

	epList := corev1.EndpointsList{
		Items: []corev1.Endpoints{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "dwb-test",
				},
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{
								IP: "1.1.1.1",
							},
						},
					},
				},
			},
		},
	}
	initTenantConfig(as3cfg, "dwb-test")
	as3 := initDefaultAS3()
	adc := as3[DeclarationKey].(as3ADC)[DefaultPartition]
	srcAdc := map[string]interface{}{}
	validateJSONAndFetchObject(adc, &srcAdc)
	tntcfg := GetTenantConfigForParttition("dwb-test")
	deltaSrc := as3ADC{}
	post := newAs3Post(&svcgrList, nil, nil, &exsvcList, nil, &epList, nil, tntcfg)
	post.generateAS3ResourceDeclaration(deltaSrc)
	body := fullResource(DefaultPartition, false, srcAdc, deltaSrc)
	printObj(body)
}

func TestMockExtenalService(t *testing.T) {
	cgRuleList := kubeovnv1alpha1.ClusterEgressRuleList{
		Items: []kubeovnv1alpha1.ClusterEgressRule{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test1",
				},
				Spec: kubeovnv1alpha1.ClusterEgressRuleSpec{
					Action: "accept",
					ExternalServices: []string{
						"exsvc1",
					},
				},
			},
		},
	}
	nsRuleList := kubeovnv1alpha1.NamespaceEgressRuleList{}
	svcRuleList := kubeovnv1alpha1.ServiceEgressRuleList{}
	exsvcList := kubeovnv1alpha1.ExternalServiceList{
		Items: []kubeovnv1alpha1.ExternalService{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exsvc1",
					Namespace: "dwb-test",
				},
				Spec: kubeovnv1alpha1.ExternalServiceSpec{
					Addresses: []string{
						"192.168.1.1",
					},
					Ports: []kubeovnv1alpha1.ExternalServicePort{
						{
							Name:     "xxx",
							Protocol: "tcp",
							Port:     "8080",
						},
					},
				},
			},
		},
	}

	as3cfg := As3Config{
		ClusterName:          "k8s",
		IsSupportRouteDomain: false,
		Tenant: []TenantConfig{
			{
				Name: "Common",
				RouteDomain: RouteDomain{
					Name: "0",
					Id:   0,
				},
				VirtualService: VirtualService{},
				Gwpool: Gwpool{
					ServerAddresses: []string{"192.168.132.1"},
				},
			},
		},
	}
	initTenantConfig(as3cfg, "dwb-test")
	tntcfg := GetTenantConfigForParttition(DefaultPartition)
	as3 := initDefaultAS3()
	adc := as3[DeclarationKey].(as3ADC)
	as3post := newAs3Post(&svcRuleList, &nsRuleList, &cgRuleList, &exsvcList, nil, nil, nil, tntcfg)
	as3post.generateAS3ResourceDeclaration(adc)
	deltaAdc := as3ADC{}
	as3post.generateAS3ResourceDeclaration(deltaAdc)
	printObj(deltaAdc)
	//delete exsvc
	body := fullResource("Common", true, adc, deltaAdc)
	printObj(body)

	//There are bwt and no bwt at the same time
	t.Log("test: There are bwt and no bwt at the same time")
	exsvcList = kubeovnv1alpha1.ExternalServiceList{
		Items: []kubeovnv1alpha1.ExternalService{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exsvc1",
					Namespace: "dwb-test",
				},
				Spec: kubeovnv1alpha1.ExternalServiceSpec{
					Addresses: []string{
						"192.168.1.1",
					},
					Ports: []kubeovnv1alpha1.ExternalServicePort{
						{
							Name:     "tcp-8080",
							Protocol: "tcp",
							Port:     "8080",
						}, {
							Name:      "tcp-443",
							Protocol:  "tcp",
							Port:      "443",
							Bandwidth: "bwc-1mbps-irule",
						},
					},
				},
			},
		},
	}
	as3post = newAs3Post(&svcRuleList, &nsRuleList, &cgRuleList, &exsvcList, nil, nil, nil, tntcfg)
	adc = as3ADC{}
	as3post.generateAS3ResourceDeclaration(adc)
	printObj(adc)

	// There are not ports, only has address
	t.Log("test: There are not ports, only has address")
	exsvcList = kubeovnv1alpha1.ExternalServiceList{
		Items: []kubeovnv1alpha1.ExternalService{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exsvc1",
					Namespace: "dwb-test",
				},
				Spec: kubeovnv1alpha1.ExternalServiceSpec{
					Addresses: []string{
						"192.168.1.1",
					},
				},
			},
		},
	}
	as3post = newAs3Post(&svcRuleList, &nsRuleList, &cgRuleList, &exsvcList, nil, nil, nil, tntcfg)
	adc = as3ADC{}
	as3post.generateAS3ResourceDeclaration(adc)
	printObj(adc)

	//There are not ports, only has address, delete exsvc
	t.Log("There are not ports, only has address, delete exsvc")
	deltaAdc1 := map[string]interface{}{}
	validateJSONAndFetchObject(adc, &deltaAdc1)
	body = fullResource(DefaultPartition, true, adc, deltaAdc1)
	printObj(body)

	//modify protocol
	printObj("modify protocol:")
	exsvcList = kubeovnv1alpha1.ExternalServiceList{
		Items: []kubeovnv1alpha1.ExternalService{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exsvc1",
					Namespace: "dwb-test",
				},
				Spec: kubeovnv1alpha1.ExternalServiceSpec{
					Addresses: []string{
						"192.168.1.1",
					},
					Ports: []kubeovnv1alpha1.ExternalServicePort{
						{
							Name:      "udp-81",
							Protocol:  "udp",
							Port:      "81-91",
							Bandwidth: "bwc-2mbps-irule",
						}, {
							Name:      "tcp-443",
							Protocol:  "tcp",
							Port:      "443",
							Bandwidth: "bwc-1mbps-irule",
						},
					},
				},
			},
		},
	}
	cgRuleList = kubeovnv1alpha1.ClusterEgressRuleList{
		Items: []kubeovnv1alpha1.ClusterEgressRule{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test1",
				},
				Spec: kubeovnv1alpha1.ClusterEgressRuleSpec{
					Action: "accept",
					ExternalServices: []string{
						"exsvc1",
						"exsvc2",
					},
				},
			},
		},
	}
	as3post = newAs3Post(nil, nil, &cgRuleList, &exsvcList, nil, nil, nil, tntcfg)
	adc = as3ADC{}
	as3post.generateAS3ResourceDeclaration(adc)
	exsvcList = kubeovnv1alpha1.ExternalServiceList{
		Items: []kubeovnv1alpha1.ExternalService{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exsvc1",
					Namespace: "dwb-test",
				},
				Spec: kubeovnv1alpha1.ExternalServiceSpec{
					Addresses: []string{
						"192.168.1.1",
					},
					Ports: []kubeovnv1alpha1.ExternalServicePort{
						{
							Name:      "tcp-81",
							Protocol:  "TCP",
							Port:      "81-91",
							Bandwidth: "bwc-2mbps-irule",
						}, {
							Name:      "tcp-443",
							Protocol:  "tcp",
							Port:      "443",
							Bandwidth: "bwc-1mbps-irule",
						},
					},
				},
			},
		},
	}
	as3post = newAs3Post(nil, nil, &cgRuleList, &exsvcList, nil, nil, nil, tntcfg)
	deltaAdc = as3ADC{}
	as3post.generateAS3ResourceDeclaration(deltaAdc)
	body = fullResource(DefaultPartition, false, adc, deltaAdc)
	printObj(body)

	//one rule , mult exsvc, delete one exsvc
	printObj("one rule , mult exsvc, delete one exsvc")
	exsvcList = kubeovnv1alpha1.ExternalServiceList{
		Items: []kubeovnv1alpha1.ExternalService{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exsvc1",
					Namespace: "dwb-test",
				},
				Spec: kubeovnv1alpha1.ExternalServiceSpec{
					Addresses: []string{
						"192.168.1.1",
					},
					Ports: []kubeovnv1alpha1.ExternalServicePort{
						{
							Name:      "udp-81",
							Protocol:  "udp",
							Port:      "81-91",
							Bandwidth: "bwc-2mbps-irule",
						}, {
							Name:      "tcp-443",
							Protocol:  "tcp",
							Port:      "443",
							Bandwidth: "bwc-1mbps-irule",
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exsvc2",
					Namespace: "dwb-test",
				},
				Spec: kubeovnv1alpha1.ExternalServiceSpec{
					Addresses: []string{
						"192.168.2.2",
					},
					Ports: []kubeovnv1alpha1.ExternalServicePort{
						{
							Name:      "udp-9090",
							Protocol:  "udp",
							Port:      "9090",
							Bandwidth: "bwc-2mbps-irule",
						},
					},
				},
			},
		},
	}
	as3post = newAs3Post(nil, nil, &cgRuleList, &exsvcList, nil, nil, nil, tntcfg)
	adc, deltaAdc = as3ADC{}, as3ADC{}
	as3post.generateAS3ResourceDeclaration(adc)
	printObj(adc)
	exsvcList = kubeovnv1alpha1.ExternalServiceList{
		Items: []kubeovnv1alpha1.ExternalService{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exsvc2",
					Namespace: "dwb-test",
				},
				Spec: kubeovnv1alpha1.ExternalServiceSpec{
					Addresses: []string{
						"192.168.2.2",
					},
					Ports: []kubeovnv1alpha1.ExternalServicePort{
						{
							Name:      "udp-9090",
							Protocol:  "udp",
							Port:      "9090",
							Bandwidth: "bwc-2mbps-irule",
						},
					},
				},
			},
		},
	}

	as3post = newAs3Post(nil, nil, &cgRuleList, &exsvcList, nil, nil, nil, tntcfg)
	as3post.generateAS3ResourceDeclaration(deltaAdc)
	printObj(deltaAdc)

	body = fullResource(DefaultPartition, true, adc, deltaAdc)
	printObj(body)
}

func TestRomteLog(t *testing.T) {
	as3cfg := As3Config{
		ClusterName:          "dwb",
		MasterCluster:        "master",
		IsSupportRouteDomain: false,
		LogPool: LogPool{
			EnableRemoteLog: false,
			LoggingEnabled:  true,
			HealthMonitor:   "udp",
			ServerAddresses: []string{"1.1.1.1:8888"},
			Template: `
{
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
}`,
		},
		Tenant: []TenantConfig{
			{
				Name:       "Common",
				Namespaces: "xxxxxxxx",
				RouteDomain: RouteDomain{
					Name: "0",
					Id:   0,
				},
				VirtualService: VirtualService{},
				Gwpool: Gwpool{
					ServerAddresses: []string{"192.168.132.1"},
				},
			},
			{
				Name:       "project1",
				Namespaces: "project1",
				RouteDomain: RouteDomain{
					Name: "rd1",
					Id:   1,
				},
				VirtualService: VirtualService{},
				Gwpool: Gwpool{
					ServerAddresses: []string{"192.101.100.1"},
				},
			}, {
				Name:       "project2",
				Namespaces: "project2",
				RouteDomain: RouteDomain{
					Name: "rd2",
					Id:   2,
				},
				VirtualService: VirtualService{},
				Gwpool: Gwpool{
					ServerAddresses: []string{"192.101.111.1"},
				},
			},
		},
	}
	initTenantConfig(as3cfg, "kube-system")
	post := &as3Post{
		tenantConfig: &as3cfg.Tenant[0],
	}
	app := as3Application{}
	post.newLogPoolDecl(app)
	printObj(app)

	as3cfg.LogPool.EnableRemoteLog = true
	initTenantConfig(as3cfg, "kube-system")
	post = &as3Post{
		tenantConfig: &as3cfg.Tenant[0],
	}
	app = as3Application{}
	post.newLogPoolDecl(app)
	printObj(app)

	as3cfg.LogPool.EnableRemoteLog = true
	as3cfg.IsSupportRouteDomain = true
	initTenantConfig(as3cfg, "kube-system")
	post = &as3Post{
		tenantConfig: &as3cfg.Tenant[1],
	}
	app = as3Application{}
	post.newLogPoolDecl(app)
	printObj(app)

	as3cfg.LogPool.EnableRemoteLog = false
	initTenantConfig(as3cfg, "kube-system")
	post = &as3Post{
		tenantConfig: &as3cfg.Tenant[1],
	}
	app = as3Application{}
	post.newLogPoolDecl(app)
	printObj(app)
}

func TestNewAddress(t *testing.T) {
	addresses := []string{
		"192.168.1.1",
		"www.baid.com",
	}
	app := as3Application{}
	newFirewallAddressList("k8s_ns_src_addr", addresses, app)
	printObj(app)

	addresses = []string{
		"192.168.1.1",
	}
	app = as3Application{}
	newFirewallAddressList("k8s_ns_src_addr", addresses, app)
	printObj(app)

	addresses = []string{
		"www.baid.com",
	}
	app = as3Application{}
	newFirewallAddressList("k8s_ns_src_addr", addresses, app)
	printObj(app)
}

func TestLogLevel(t *testing.T) {
	klog.InitFlags(nil)
	flag.Set("v", "3")
	klog.Infof("xxx")
	klog.V(1).Infof("low level log")
	klog.V(4).Infof("high level log")
	klog.Errorf("error log")
}

func TestSupportRouteDomain(t *testing.T) {

	svcgrList := kubeovnv1alpha1.ServiceEgressRuleList{
		Items: []kubeovnv1alpha1.ServiceEgressRule{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svcgr",
					Namespace: "dwb-test",
				},
				Spec: kubeovnv1alpha1.ServiceEgressRuleSpec{
					Action:           "accept",
					Logging:          true,
					Service:          "test",
					ExternalServices: []string{"exsvc"},
				},
			},
		},
	}
	exsvcList := kubeovnv1alpha1.ExternalServiceList{
		Items: []kubeovnv1alpha1.ExternalService{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exsvc",
					Namespace: "dwb-test",
				},
				Spec: kubeovnv1alpha1.ExternalServiceSpec{
					Addresses: []string{
						"192.168.2.2",
					},
					Ports: []kubeovnv1alpha1.ExternalServicePort{
						{
							Name:      "udp-9090",
							Protocol:  "udp",
							Port:      "9090",
							Bandwidth: "bwc-2mbps-irule",
						},
					},
				},
			},
		},
	}

	as3cfg := As3Config{
		ClusterName:          "cck8s",
		IsSupportRouteDomain: true,
		Tenant: []TenantConfig{
			{
				Name: "Common",
				RouteDomain: RouteDomain{
					Name: "0",
					Id:   0,
				},
				VirtualService: VirtualService{},
				Gwpool: Gwpool{
					ServerAddresses: []string{"192.168.132.1"},
				},
			},
			{
				Name:       "dwb-test",
				Namespaces: "dwb-test",
				RouteDomain: RouteDomain{
					Name: "rd1",
					Id:   1,
				},
				VirtualService: VirtualService{
					VirtualAddresses: VirtualAddresses{
						VirtualAddress: "1.1.1.1",
						IcmpEcho:       "disable",
						ArpEnabled:     true,
					},
				},
				Gwpool: Gwpool{
					ServerAddresses: []string{"192.168.132.2"},
				},
			},
		},
	}

	epList := corev1.EndpointsList{
		Items: []corev1.Endpoints{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "dwb-test",
				},
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{
								IP: "1.1.1.1",
							},
						},
					},
				},
			},
		},
	}
	initTenantConfig(as3cfg, "dwb-test")
	tntcfg := GetTenantConfigForParttition("dwb-test")
	//as3 := initDefaultAS3()
	adc := as3ADC{}
	printObj(adc)
	post := newAs3Post(&svcgrList, nil, nil, &exsvcList, nil, &epList, nil, tntcfg)
	delta := as3ADC{}
	post.generateAS3ResourceDeclaration(delta)
	printObj(delta)
	body := fullResource("dwb-test", false, adc, delta)
	printObj(body)
}
