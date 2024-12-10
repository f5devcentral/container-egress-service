package as3

const (
	// patch operations
	OpAdd     = "add"
	OpRemove  = "remove"
	OpReplace = "replace"
)

const (
	//ADC class
	classAS3         = "AS3"
	ClassADC         = "ADC"
	ClassApplication = "Application"
	ClassTenant      = "Tenant"

	// AS3 classes
	ClassFirewallAddressList  = "Firewall_Address_List"
	ClassFirewallPortList     = "Firewall_Port_List"
	ClassFirewallRuleList     = "Firewall_Rule_List"
	ClassFirewallPolicy       = "Firewall_Policy"
	ClassVirtualServerL4      = "Service_L4"
	ClassServiceAddress       = "Service_Address"
	ClassPoll                 = "Pool"
	ClassSecurityLogProfile   = "Security_Log_Profile"
	ClassLogPublisher         = "Log_Publisher"
	ClassLogDestination       = "Log_Destination"
	ClassNatPolicy            = "NAT_Policy"
	ClassNatSourceTranslation = "NAT_Source_Translation"
)

const (
	ClassKey                  = "class"
	TemplateKey               = "template"
	SharedKey                 = "Shared"
	CommonKey                 = "Common"
	DeclarationKey            = "declaration"
	EnforcedPolicyKey         = "enforcedPolicy"
	FwEnforcedPolicyKey       = "fwEnforcedPolicy"
	DefaultRouteDomainKey     = "defaultRouteDomain"
	PolicyFirewallEnforcedKey = "policyFirewallEnforced"

	SharedValue      = "shared"
	TenantValue      = "Tenant"
	ApplicationValue = "Application"
	DefaultPartition = CommonKey

	allNamespace = "__all__"
)

const (
	RuleTypeLabel = "cpaas.io/ruleType"

	RuleTypeGlobal    = "global"
	RuleTypeNamespace = "namespace"
	RuleTypeService   = "service"
)

const (
	DenyAllRuleName = "deny_all_rule"
)

const (
	NamespaceCidr = "ovn.kubernetes.io/cidr"
)

// eg: Common/Shared/k8s
const pathProfix = "/%s/Shared/%s"
