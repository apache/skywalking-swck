module github.com/apache/skywalking-swck

go 1.13

require (
	github.com/apache/skywalking-cli v0.0.0-20201121070253-814817cd2396
	github.com/evanphx/json-patch v4.9.0+incompatible
	github.com/go-logr/logr v0.2.1
	github.com/kubernetes-sigs/custom-metrics-apiserver v0.0.0-20201110135240-8c12d6d92362
	github.com/machinebox/graphql v0.2.2
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.0.0
	github.com/urfave/cli v1.22.1
	k8s.io/api v0.19.3
	k8s.io/apimachinery v0.19.3
	k8s.io/apiserver v0.19.3
	k8s.io/client-go v0.19.3
	k8s.io/component-base v0.19.3
	k8s.io/klog/v2 v2.3.0
	k8s.io/metrics v0.19.3
	sigs.k8s.io/controller-runtime v0.7.0-alpha.6
)

replace github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1
