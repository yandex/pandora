module github.com/yandex/pandora

go 1.23.8

require (
	github.com/PaesslerAG/jsonpath v0.1.1
	github.com/antchfx/htmlquery v1.3.2
	github.com/antchfx/xpath v1.3.1
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2
	github.com/c2h5oh/datasize v0.0.0-20220606134207-859f65c6625b
	github.com/facebookgo/stack v0.0.0-20160209184415-751773369052
	github.com/facebookgo/stackerr v0.0.0-20150612192056-c2fcf88613f4
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang/protobuf v1.5.4
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/hcl/v2 v2.23.0
	github.com/jhump/protoreflect v1.15.6
	github.com/json-iterator/go v1.1.12
	github.com/mitchellh/mapstructure v1.5.1-0.20220423185008-bf980b35cac4
	github.com/pkg/errors v0.9.1
	github.com/spf13/afero v1.12.0
	github.com/spf13/viper v1.20.1
	github.com/stretchr/testify v1.10.0
	github.com/zclconf/go-cty v1.13.2
	go.uber.org/atomic v1.11.0
	go.uber.org/zap v1.27.0
	golang.org/x/exp v0.0.0-20240707233637-46b078467d37
	golang.org/x/net v0.35.0
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da
	google.golang.org/grpc v1.71.0
	google.golang.org/protobuf v1.36.5
	gopkg.in/bluesuncorp/validator.v9 v9.10.0
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/PaesslerAG/gval v1.2.1 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/apparentlymart/go-textseg/v13 v13.0.0 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/bufbuild/protocompile v0.9.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-test/deep v1.1.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	github.com/sagikazarmark/locafero v0.7.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.opentelemetry.io/otel v1.35.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.35.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/mod v0.23.0 // indirect
	golang.org/x/sync v0.11.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	golang.org/x/tools v0.30.1-0.20250221230316-5055f70f240c // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

exclude github.com/keybase/go.dbus v0.0.0-20220506165403-5aa21ea2c23a

replace github.com/insomniacslk/dhcp => github.com/insomniacslk/dhcp v0.0.0-20210120172423-cc9239ac6294

replace cloud.google.com/go/pubsub => cloud.google.com/go/pubsub v1.30.0

replace github.com/grpc-ecosystem/grpc-gateway/v2 => github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.1

replace go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc => go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.22.0

replace github.com/jackc/pgtype => github.com/jackc/pgtype v1.12.0

replace github.com/aws/aws-sdk-go => github.com/aws/aws-sdk-go v1.46.7

replace k8s.io/api => k8s.io/api v0.26.1

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.26.1

replace k8s.io/apimachinery => k8s.io/apimachinery v0.26.1

replace k8s.io/apiserver => k8s.io/apiserver v0.26.1

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.26.1

replace k8s.io/client-go => k8s.io/client-go v0.26.1

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.26.1

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.26.1

replace k8s.io/code-generator => k8s.io/code-generator v0.26.1

replace k8s.io/component-base => k8s.io/component-base v0.26.1

replace k8s.io/cri-api => k8s.io/cri-api v0.23.5

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.26.1

replace k8s.io/dynamic-resource-allocation => k8s.io/dynamic-resource-allocation v0.26.1

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.26.1

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.26.1

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.26.1

replace k8s.io/kubelet => k8s.io/kubelet v0.26.1

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.26.1

replace k8s.io/mount-utils => k8s.io/mount-utils v0.26.2-rc.0

replace k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.26.1

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.26.1

replace github.com/temporalio/features => github.com/temporalio/features v0.0.0-20231218231852-27c681667dae

replace github.com/temporalio/features/features => github.com/temporalio/features/features v0.0.0-20231218231852-27c681667dae

replace github.com/temporalio/features/harness/go => github.com/temporalio/features/harness/go v0.0.0-20231218231852-27c681667dae

replace github.com/temporalio/omes => github.com/temporalio/omes v0.0.0-20240429210145-5fa5c107b7a8

replace github.com/aleroyer/rsyslog_exporter => github.com/prometheus-community/rsyslog_exporter v1.1.0

replace github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.20.5

replace github.com/prometheus/client_model => github.com/prometheus/client_model v0.6.1

replace github.com/prometheus/common => github.com/prometheus/common v0.62.0

replace github.com/distribution/reference => github.com/distribution/reference v0.5.0

replace github.com/jackc/pgconn => github.com/jackc/pgconn v1.14.0

replace github.com/jackc/pgproto3/v2 => github.com/jackc/pgproto3/v2 v2.3.2

replace github.com/nexus-rpc/sdk-go => github.com/nexus-rpc/sdk-go v0.0.7

replace github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.14.24
