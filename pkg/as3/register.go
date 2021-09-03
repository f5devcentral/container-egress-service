package as3

import "sync"

var syncMap sync.Map

const (
	as3DefaultTemplateKey = "__AS3_DEFAULT_TEMPLATE__"
	as3CommonPartitionKey = "__common__"
	currentClusterKey = "__CLUSTER__"
	isSupportRouteDomainKey = "__IS_SUPPORT_ROUTE_DOMAIN__"

)

func registValue(name, v interface{}){
	syncMap.Store(name, v)
}

func getValue(name string) interface{}{
	v, ok := syncMap.Load(name)
	if ok {
		return v
	}
	return nil
}
