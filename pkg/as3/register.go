package as3

import "sync"

var syncMap sync.Map

const (
	as3DefaultTemplateKey     = "__AS3_DEFAULT_TEMPLATE__"
	currentClusterKey         = "__CLUSTER__"
	isSupportRouteDomainKey   = "__IS_SUPPORT_ROUTE_DOMAIN__"
	logPoolKey                = "__LOG_POOL__"
	schemaVersionKey          = "__SCHEMAVERSION__"
	namespaceCacheKey         = "__NAMESPACE_CACHE_KEY__"
	partitionCacheKey         = "__PARTITION_CACHE_KEY__"
	masterClusterKey          = "__MASTER_CLUSTER__"
	as3IRulesListKey          = "__AS3_IRULES_LIST_KEY__"
	clusterSvcExtNamespaceKey = "__CLUSTER_SVC_EXT_NAMESPACE__"
	externalIPAddressesKey    = "__EXTERNAL_IP_ADDRESSES__"
)

func registValue(name, v interface{}) {
	syncMap.Store(name, v)
}

func getValue(name string) interface{} {
	v, ok := syncMap.Load(name)
	if ok {
		return v
	}
	return nil
}
