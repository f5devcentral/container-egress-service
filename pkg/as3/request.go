package as3

import (
	"fmt"
	"sync"
	"time"

	"github.com/kubeovn/ces-controller/pkg/apis/kubeovn.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

func (c *Client) As3Request(serviceEgressList *v1alpha1.ServiceEgressRuleList, namespaceEgressList *v1alpha1.NamespaceEgressRuleList,
	clusterEgressList *v1alpha1.ClusterEgressRuleList, externalServiceList *v1alpha1.ExternalServiceList,
	endpointList *corev1.EndpointsList, namespaceList *corev1.NamespaceList, tenantConfig *TenantConfig,
	ty string, isDelete bool) error {

	as3PostParam := newAs3Post(serviceEgressList, namespaceEgressList, clusterEgressList, externalServiceList, endpointList,
		namespaceList, tenantConfig)
	deltaAdc := as3ADC{}
	as3PostParam.generateAS3ResourceDeclaration(deltaAdc)
	partition := tenantConfig.Name
	adcStr, err := c.Get(partition)
	if err != nil {
		if isNotFound(err){
			adcStr = "{}"
		}else{
			return fmt.Errorf("failed to get tenant[%s], error: %v", partition, err)
		}
	}
	srcAdc := map[string]interface{}{}
	err = validateJSONAndFetchObject(adcStr, &srcAdc)
	if err != nil {
		return err
	}
	reqBody := fullResource(partition, isDelete, srcAdc, deltaAdc)
	err = c.post(reqBody, partition)
	if err != nil {
		err = fmt.Errorf("failed to request AS3 POST API: %v", err)
		return err
	}

	if ty == RuleTypeGlobal {
		//get route domian police
		globalPolicyPath := getAs3UsePathForPartition(partition, getAs3PolicyAttr(RuleTypeGlobal, tenantConfig.RouteDomain.Name))
		url := "/mgmt/tm/security/firewall/global-rules"
		response, err := c.getF5Resource(url)
		if err != nil {
			return err
		}
		isExist := false
		if val, ok := response[EnforcedPolicyKey]; ok {
			fwEnforcedPolicy := val.(string)
			if fwEnforcedPolicy == globalPolicyPath {
				isExist = true
			}
		}
		// created global policy
		if !isExist {
			globalPolicy := map[string]string{
				EnforcedPolicyKey: globalPolicyPath,
			}
			err := c.patchF5Reource(globalPolicy, url)
			if err != nil {
				return err
			}

			err = c.storeDisk()
			if err != nil {
				return err
			}
		}

	} else if ty == RuleTypeNamespace {
		nsRouteDomainPolicePath := getAs3UsePathForPartition(partition, getAs3PolicyAttr("ns", tenantConfig.RouteDomain.Name))
		//get route domian police
		url := fmt.Sprintf("/mgmt/tm/net/route-domain/~%s~%s", tenantConfig.Name, tenantConfig.RouteDomain.Name)
		response, err := c.getF5Resource(url)
		if err != nil {
			klog.Errorf("failed to get route domian %s, error:%v", tenantConfig.RouteDomain.Name, err)
			return err
		}

		isNsPolicyExist := false
		if val, ok := response[FwEnforcedPolicyKey]; ok {
			fwEnforcedPolicy := val.(string)
			if fwEnforcedPolicy == nsRouteDomainPolicePath {
				isNsPolicyExist = true
			}
		}
		if !isNsPolicyExist {
			// binding route domain policy
			nsPolicy := map[string]string{
				FwEnforcedPolicyKey: nsRouteDomainPolicePath,
			}
			err := c.patchF5Reource(nsPolicy, url)
			if err != nil {
				return err
			}

			err = c.storeDisk()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Client) UpdateBigIPSourceAddress(addrList BigIpAddressList, tntcfg *TenantConfig, namespace, ruleName, svcName string) error {
	srcAddressAttr := getAs3SrcAddressAttr("svc", namespace, ruleName, svcName)
	url := fmt.Sprintf("/mgmt/tm/security/firewall/address-list/~%s~Shared~%s", tntcfg.Name, srcAddressAttr)
	if tntcfg.RouteDomain.Id != 0 {
		for k := range addrList.Addresses {
			addrList.Addresses[k].Name = addrList.Addresses[k].Name + "%10"
		}
	}
	err := c.patchF5Reource(addrList, url)
	if err != nil {
		err = fmt.Errorf("failed to request BIG-IP Patch API: %v", err)
		return err
	}

	go c.frequency()
	return nil
}

type syncFrequency struct {
	updateTimes []time.Time
	lock        sync.Mutex
}

var syncFq = syncFrequency{}

func (c *Client) frequency() {
	syncFq.lock.Lock()
	defer syncFq.lock.Unlock()
	now := time.Now()
	isUpdateEpFq := func() bool {
		times := 0
		for _, v := range syncFq.updateTimes {
			if times > 5 {
				return true
			}
			if now.Sub(v) < 2*60*time.Second {
				times += 1
			}
		}
		return false
	}
	if len(syncFq.updateTimes) > 10 || isUpdateEpFq() {
		err := c.storeDisk()
		if err != nil {
			klog.Errorf("BIG-IP store disk error: %v", err)
			return
		}
		syncFq.updateTimes = []time.Time{}
	} else {
		syncFq.updateTimes = append(syncFq.updateTimes, now)
	}
}
