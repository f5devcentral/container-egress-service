package as3

import (
	"fmt"
	"io/ioutil"
	"strings"

	"sigs.k8s.io/yaml"
)

func InitAs3Tenant(client *Client, filePath string, initialized bool) error {
	configData, err := ioutil.ReadFile(filePath + "/ces-conf.yaml")
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %v", filePath, err)
	}
	var as3Config As3Config
	err = yaml.Unmarshal(configData, &as3Config)
	if err != nil {
		return err
	}

	//store tenant in in sync.Map
	initTenantConfig(as3Config)

	nsConfig := GetTenantConfigForParttition(DefaultPartition)
	if nsConfig == nil {
		msg := `
namespaces:
  ##common partiton config, init AS3 needs
  - name: "Common"
    parttion: Common
    virtualService:
      template: ''
    gwPool:
      serverAddresses:
        - "192.168.10.1"
`
		return fmt.Errorf("No configured Common, please configured, eg: \n%s\n", msg)

	}

	if !initialized {
		return client.post(initDefaultAS3(), DefaultPartition)
	}

	return nil
}

func initTenantConfig(as3Config As3Config) {
	//store cluster in sync.Map
	registValue(schemaVersionKey, as3Config.SchemaVersion)
	registValue(currentClusterKey, as3Config.ClusterName)
	if as3Config.MasterCluster == "" {
		registValue(masterClusterKey, as3Config.ClusterName)
	} else {
		registValue(masterClusterKey, as3Config.MasterCluster)
	}
	registValue(isSupportRouteDomainKey, as3Config.IsSupportRouteDomain)
	registValue(logPoolKey, as3Config.LogPool)
	//store tenant in in sync.Map
	for _, tntconf := range as3Config.Tenant {
		if tntconf.Name == DefaultPartition {
			tntconf.RouteDomain = RouteDomain{
				Id:   0,
				Name: "0",
			}
		}
		cacheTenantConfigForParttition(tntconf.Name, tntconf)
		for _, ns := range strings.Split(tntconf.Namespaces, ",") {
			cacheTenantConfigForNamespace(ns, tntconf)
		}
	}
}

func cacheTenantConfigForParttition(partition string, tntcfg TenantConfig) {
	v := getValue(partitionCacheKey)
	if v == nil {
		registValue(partitionCacheKey, map[string]*TenantConfig{})
	}
	v = getValue(partitionCacheKey)
	cacheMap := v.(map[string]*TenantConfig)
	cacheMap[partition] = &tntcfg
	registValue(partitionCacheKey, cacheMap)
}

func GetTenantConfigForParttition(partition string) *TenantConfig {
	v := getValue(partitionCacheKey)
	if v == nil {
		return nil
	}
	cacheMap, ok := v.(map[string]*TenantConfig)
	if !ok {
		return nil
	}
	if !IsSupportRouteDomain() {
		return cacheMap[DefaultPartition]
	}
	tntcfg, ok := cacheMap[partition]
	if !ok {
		return nil
	}

	return tntcfg
}

func cacheTenantConfigForNamespace(namespace string, tntcfg TenantConfig) {
	v := getValue(namespaceCacheKey)
	if v == nil {
		registValue(namespaceCacheKey, map[string]*TenantConfig{})
	}
	v = getValue(namespaceCacheKey)
	cacheMap := v.(map[string]*TenantConfig)
	cacheMap[namespace] = &tntcfg
	registValue(namespaceCacheKey, cacheMap)
}

func GetTenantConfigForNamespace(namespace string) *TenantConfig {
	v := getValue(namespaceCacheKey)
	if v == nil {
		return nil
	}
	cacheMap, ok := v.(map[string]*TenantConfig)
	if !ok {
		return nil
	}
	tntcfg, ok := cacheMap[namespace]
	if !ok {
		return nil
	}
	if !IsSupportRouteDomain() {
		return GetTenantConfigForParttition(DefaultPartition)
	}
	return tntcfg
}

func GetDefaultTemplate() map[string]interface{} {
	v := getValue(as3DefaultTemplateKey)
	return v.(map[string]interface{})
}

func GetCluster() string {
	v := getValue(currentClusterKey)
	return v.(string)
}

func getLogPool() LogPool {
	v := getValue(logPoolKey)
	if v == nil {
		return LogPool{}
	}
	return v.(LogPool)
}

func getMasterCluster() string {
	v := getValue(masterClusterKey)
	if v == nil {
		return GetCluster()
	}
	return v.(string)
}

func IsSupportRouteDomain() bool {
	v := getValue(isSupportRouteDomainKey)
	return v.(bool)
}

func IsNotFound(err error) bool {
	if strings.Contains(err.Error(), "status code 404") {
		return true
	}
	return false
}

func getSchemaVersion() string {
	v := getValue(schemaVersionKey)
	if v == nil {
		return "3.29.0"
	}
	return v.(string)
}

func skipDeleteShareApplicationClassOrAttr(attr string) bool {
	skipDeleteShareApplicationAttr := map[string]bool{
		ClassKey:                 true,
		TemplateKey:              true,
		getAs3VSAttr():           true,
		getAs3GwPoolAttr():       true,
		getAllDenyRuleListAttr(): true,
	}
	shareApp := as3Application{}
	ac := &as3Post{}
	ac.newLogPoolDecl(shareApp)
	for k, _ := range shareApp {
		skipDeleteShareApplicationAttr[k] = true
	}
	return skipDeleteShareApplicationAttr[attr]
}
