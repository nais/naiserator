module github.com/nais/naiserator

require (
	github.com/Shopify/sarama v1.28.0
	github.com/ghodss/yaml v1.0.0
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.1.2
	github.com/hashicorp/go-multierror v1.0.0
	github.com/imdario/mergo v0.3.12
	github.com/klauspost/compress v1.12.2 // indirect
	github.com/magiconair/properties v1.8.5
	github.com/mitchellh/hashstructure v1.1.0
	github.com/mitchellh/mapstructure v1.4.1
	github.com/nais/liberator v0.0.0-20210825131439-d28ee52da1a0
	github.com/novln/docker-parser v0.0.0-20190306203532-b3f122c6978e
	github.com/prometheus/client_golang v1.11.0
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/net v0.0.0-20210503060351-7fd8e65b6420 // indirect
	google.golang.org/protobuf v1.26.0
	gopkg.in/yaml.v2 v2.4.0
	istio.io/api v0.0.0-20210809175348-eff556fb5d8a
	istio.io/client-go v1.11.2
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v0.21.3
	k8s.io/utils v0.0.0-20210722164352-7f3ee0f31471
	sigs.k8s.io/controller-runtime v0.9.5
)

go 1.15

replace github.com/nais/liberator => ../liberator
