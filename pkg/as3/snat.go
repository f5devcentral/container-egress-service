package as3

import "encoding/json"

const defaultSnatTranslation = "k8s_snat_default_translation"
const defaultSnatRule = "k8s_snat_automap"
const defaultSnatPolicy = "k8s_snat_policy"

type Destination struct {
	AddressLists []Use `json:"addressLists,omitempty"`
	PortLists    []Use `json:"portLists,omitempty"`
}

type Source struct {
	AddressLists []Use `json:"addressLists"`
}

type NatRule struct {
	Name              string       `json:"name"`
	Protocol          string       `json:"protocol"`
	Source            *Source      `json:"source,omitempty"`
	Destination       *Destination `json:"destination,omitempty"`
	SourceTranslation Use          `json:"sourceTranslation"`
}

type NatPolicy struct {
	Class string    `json:"class"`
	Rules []NatRule `json:"rules"`
}

func (np *NatPolicy) String() string {
	data, err := json.Marshal(np)
	if err != nil {
		return "{}"
	}

	return string(data)
}

func ParseStringToSnatPolicy(data string) (*NatPolicy, error) {
	var ret NatPolicy
	if err := json.Unmarshal([]byte(data), &ret); err != nil {
		return nil, err
	}
	return &ret, nil
}

type NatSourceTranslation struct {
	Class     string   `json:"class"`
	Type      string   `json:"type"`
	Addresses []string `json:"addresses"`
	Ports     []string `json:"ports"`
}

func newNatSourceTranslation(attr string, ips []string, shareApp as3Application) {
	shareApp[attr] = NatSourceTranslation{
		Class:     ClassNatSourceTranslation,
		Type:      "dynamic-pat",
		Addresses: ips,
		Ports:     []string{"10000-50000"},
	}
}
