package as3

import (
	"encoding/json"
	"reflect"
	"testing"
)

// go test -mod=vendor -run="^TestSnatPolicyString$" -v
func TestSnatPolicyString(t *testing.T) {
	sp := NatPolicy{
		Class: "NAT_Policy",
		Rules: []NatRule{
			{
				Destination: &Destination{
					AddressLists: []Use{
						{
							Use: "/Common/Shared/k8s_snat_ces_busybox-snat_ext_busybox-svc_address-name",
						},
					},
					PortLists: []Use{
						{
							Use: "/Common/Shared/k8s_snat_ces_busybox-snat_ext_busybox-svc_ports_tcp-name",
						},
					},
				},
				Name:     "busybox_snat",
				Protocol: "tcp",
				Source: &Source{
					AddressLists: []Use{
						{
							Use: "/Common/Shared/k8s_snat_ces_busybox-snat_ep_busybox-svc_src_address",
						},
					},
				},
				SourceTranslation: Use{
					Use: "/Common/Shared/k8s_snat_ces_busybox-snat_source_translation",
				},
			},
			{
				Name:     "k8s_snat_automap",
				Protocol: "any",
				SourceTranslation: Use{
					Use: "automap",
				},
			},
		},
	}

	t.Log(sp.String())
}

// go test -mod=vendor -run="^TestParseStringToSnatPolicy$" -v
func TestParseStringToSnatPolicy(t *testing.T) {
	const str = `{"class":"NAT_Policy","rules":[{"destination":{"addressLists":[{"use":"/Common/Shared/k8s_snat_ces_busybox-snat_ext_busybox-svc_address-name"}],"portLists":[{"use":"/Common/Shared/k8s_snat_ces_busybox-snat_ext_busybox-svc_ports_tcp-name"}]},"name":"busybox_snat","protocol":"tcp","source":{"addressLists":[{"use":"/Common/Shared/k8s_snat_ces_busybox-snat_ep_busybox-svc_src_address"}]},"sourceTranslation":{"use":"/Common/Shared/k8s_snat_ces_busybox-snat_source_translation"}},{"destination":{"addressLists":null,"portLists":null},"name":"k8s_snat_automap","protocol":"any","source":{"addressLists":null},"sourceTranslation":{"use":"automap"}}]}`
	policy, err := ParseStringToSnatPolicy(str)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%#v", policy)
}

// go test -mode=vendor -run="TestMarshalJSONSnatPolicy" -v
func TestMarshalJSONSnatPolicy(t *testing.T) {
	sp := NatPolicy{
		Class: "NAT_Policy",
		Rules: []NatRule{
			{
				Name:     "test",
				Protocol: "tcp",
			},
		},
	}

	data, err := json.Marshal(sp)
	if err != nil {
		t.Fatal(err)
	}
	println(string(data))
	if !reflect.DeepEqual(string(data), `{"class":"NAT_Policy","rules":[{"name":"test","protocol":"tcp"}]}`) {
		t.Fatal("json marshal error")
	}
}
