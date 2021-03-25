module github.com/apache/skywalking-swck

go 1.14

require (
	github.com/Masterminds/sprig/v3 v3.1.0
	github.com/apache/skywalking-cli v0.0.0-20210209032327-04a0ce08990f
	github.com/evanphx/json-patch v4.9.0+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.3.0
	github.com/kubernetes-sigs/custom-metrics-apiserver v0.0.0-20201110135240-8c12d6d92362
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.6.1
	github.com/urfave/cli v1.22.1
	k8s.io/api v0.20.1
	k8s.io/apimachinery v0.20.1
	k8s.io/apiserver v0.20.1
	k8s.io/client-go v0.20.1
	k8s.io/component-base v0.20.1
	k8s.io/klog/v2 v2.4.0
	k8s.io/metrics v0.20.1
	sigs.k8s.io/controller-runtime v0.7.0
)

replace github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1

replace skywalking/network => github.com/apache/skywalking-cli/gen-codes/skywalking/network v0.0.0-20210209032327-04a0ce08990f

replace google.golang.org/grpc => google.golang.org/grpc v1.29.1
