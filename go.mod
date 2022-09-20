module github.com/kubeovn/ces-controller

go 1.15

require (
	github.com/emicklei/go-restful v2.16.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/spf13/viper v1.8.1
	golang.org/x/text v0.3.7 // indirect
	k8s.io/api v0.20.6
	k8s.io/apimachinery v0.20.6
	k8s.io/client-go v0.20.6
	k8s.io/code-generator v0.20.6
	k8s.io/gengo v0.0.0-20210813121822-485abfe95c7c // indirect
	k8s.io/klog/v2 v2.10.0
	k8s.io/kube-openapi v0.0.0-20210817084001-7fbd8d59e5b8 // indirect
)

replace (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible => github.com/golang-jwt/jwt v3.2.1+incompatible
	github.com/miekg/dns v1.0.14 => github.com/miekg/dns v1.1.25
	golang.org/x/crypto v0.0.0-20181029021203-45a5f77698d3 => golang.org/x/crypto v0.0.0-20220314234659-1baeb1ce4c0b
	golang.org/x/net v0.0.0-20211112202133-69e39bad7dc2 => golang.org/x/net v0.0.0-20211209124913-491a49abca63
	golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1 => golang.org/x/sys v0.0.0-20220412211240-33da011f77ad
)
