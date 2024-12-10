package as3

import (
	"fmt"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"k8s.io/klog/v2"
)

func InitAs3Tenant(client *Client, filePath string, cesNamespace string) error {
	config := viper.New()
	config.AddConfigPath(filePath)
	config.SetConfigName("ces-conf")
	config.SetConfigType("yaml")
	// 读取该配置文件
	config.ReadInConfig()
	config.WatchConfig()
	initConfigFunc := func() {
		var as3Config As3Config
		err := config.Unmarshal(&as3Config)
		if err != nil {
			panic(fmt.Sprintf(" yaml unmarshal err: %v", err))
		}
		initTenantConfig(as3Config, cesNamespace)
	}
	initConfigFunc()
	config.OnConfigChange(func(in fsnotify.Event) {
		klog.Info("file[ces-conf.yaml] has been modified, configuration reinitialization !")
		go initConfigFunc()
	})

	nsConfig := GetTenantConfigForParttition(DefaultPartition)
	if nsConfig == nil {
		msg := `
namespaces:
  ##common partiton config, init AS3 needs
  - name: "Common"
    virtualService:
      template: ''
    gwPool:
      serverAddresses:
        - "192.168.10.1"
`
		return fmt.Errorf("No configured Common, please configured, eg: \n%s\n", msg)
	}

	if getMasterCluster() == GetCluster() {
		as3Str, err := client.Get(DefaultPartition)
		if err != nil {
			return fmt.Errorf("failed to get partition, due to: %v", err)
		}
		if as3Str == "{}" {
			return client.post(initDefaultAS3(), DefaultPartition)
		}
	}
	return nil
}

func initTenantConfig(as3Config As3Config, cesNamespace string) {
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
	registValue(as3IRulesListKey, as3Config.IRule)
	//store ces serviceacount namespace, used cluster exsvc ns
	registValue(clusterSvcExtNamespaceKey, cesNamespace)
	//store external ip addresses
	registValue(externalIPAddressesKey, as3Config.ExternalIPAddresses)
	//store tenant in in sync.Map
	for _, tntconf := range as3Config.Tenant {
		if tntconf.Name == DefaultPartition {
			tntconf.RouteDomain = RouteDomain{
				Id:   0,
				Name: "0",
			}
		}
		cacheTenantConfigForParttition(tntconf.Name, tntconf)
		if strings.TrimSpace(tntconf.Namespaces) == "" {
			continue
		}
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

func getIRules() []string {
	v := getValue(as3IRulesListKey)
	if v == nil {
		return []string{}
	}
	irules, ok := v.([]string)
	if !ok {
		return []string{}
	}
	return irules
}

func getSchemaVersion() string {
	v := getValue(schemaVersionKey)
	if v == nil {
		return "3.29.0"
	}
	return v.(string)
}

func isConfigLogProfile() bool {
	if !getLogPool().LoggingEnabled || getLogPool().Template == "" {
		return false
	}
	return true
}

func skipDeleteShareApplicationClassOrAttr(partition, attr string) bool {
	skipDeleteShareApplicationAttr := map[string]bool{
		ClassKey:                 true,
		TemplateKey:              true,
		getAs3VSAttr():           true,
		getAs3VsVaAttr():         true,
		getAs3GwPoolAttr():       true,
		getAllDenyRuleListAttr(): true,
	}
	shareApp := as3Application{}
	tntcfg := GetTenantConfigForParttition(partition)
	ac := newAs3Post(nil, nil, nil, nil, nil, nil, nil, tntcfg)
	ac.newLogPoolDecl(shareApp)
	for k, _ := range shareApp {
		skipDeleteShareApplicationAttr[k] = true
	}
	return skipDeleteShareApplicationAttr[attr]
}

func GetIRules() string {
	irules := getIRules()
	return strings.Join(irules, ",")
}

func GetClusterSvcExtNamespace() string {
	clusterSvcExtNamespace := getValue(clusterSvcExtNamespaceKey)
	if clusterSvcExtNamespace == nil {
		return "kube-system"
	}
	return clusterSvcExtNamespace.(string)
}

func GetExternalIPAddresses() []string {
	v := getValue(externalIPAddressesKey)
	return v.([]string)
}
