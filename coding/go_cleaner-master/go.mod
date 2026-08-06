module code.byted.org/analyzers/go_cleaner

go 1.19
// 官方的thriftgo不提供方法的位置信息，所以需要替换成自己的魔改版本
// replace github.com/cloudwego/thriftgo => ../thriftgo

require (
	code.byted.org/codebase/analysis-golang-engine v0.0.0-20230808110938-62e77b8d4a0b
	code.byted.org/codebase/analysis-model v0.0.0-20220708094922-27597a1d2c1d
	code.byted.org/dp/gotqs v1.0.0
	code.byted.org/gdp/log v0.9.5
	code.byted.org/gopkg/env v1.5.22
	code.byted.org/gopkg/logs v1.2.21
	code.byted.org/gopkg/metrics v1.4.25
	code.byted.org/gopkg/tccclient v1.6.0
	code.byted.org/gorm/bytedgorm v0.9.4
	code.byted.org/lang/gg v0.21.0
	code.byted.org/middleware/hertz v1.10.9
	code.byted.org/middleware/hertz_ext/v2 v2.1.6
	code.byted.org/ucenter/bdsso_sessionlib v1.1.1-rc.1
	code.byted.org/wujianhui.0218/thriftgo v0.3.18-patch.3 // 官方的thriftgo不提供方法的位置信息，所以需要替换成自己的魔改版本，go install不支持replace
	github.com/atotto/clipboard v0.1.4
	github.com/bytedance/gopkg v0.0.0-20230728082804-614d0af6619b
	github.com/masatana/go-textdistance v0.0.0-20191005053614-738b0edac985
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.7.0
	golang.org/x/tools v0.18.0
	google.golang.org/protobuf v1.34.2
	gorm.io/gorm v1.25.4
)

require (
	github.com/bufbuild/protocompile v0.14.1
	github.com/cloudwego/hertz v0.6.7
	golang.org/x/term v0.20.0
)

require (
	code.byted.org/aiops/apm_vendor_byted v0.0.22 // indirect
	code.byted.org/aiops/metrics_codec v0.0.18 // indirect
	code.byted.org/aiops/monitoring-common-go v0.0.3 // indirect
	code.byted.org/bytedtrace/bytedtrace-client-go v1.0.46 // indirect
	code.byted.org/bytedtrace/bytedtrace-common/go v0.0.12 // indirect
	code.byted.org/bytedtrace/bytedtrace-compatible-client-go v0.0.14 // indirect
	code.byted.org/bytedtrace/bytedtrace-conf-provider-client-go v0.0.19 // indirect
	code.byted.org/bytedtrace/interface-go v1.0.20 // indirect
	code.byted.org/bytedtrace/serializer-go v1.0.0 // indirect
	code.byted.org/codebase/go-log v0.0.0-20210507122058-66a5ff9c3095 // indirect
	code.byted.org/duanyi.aster/gopkg v0.0.4 // indirect
	code.byted.org/gdp/env v0.7.4 // indirect
	code.byted.org/gin/ginex v1.8.0 // indirect
	code.byted.org/gopkg/apm_vendor_interface v0.0.2 // indirect
	code.byted.org/gopkg/asyncache v0.0.0-20210129072708-1df5611dba17 // indirect
	code.byted.org/gopkg/asynccache v0.0.0-20201112072351-d630cb60c767 // indirect
	code.byted.org/gopkg/bytedmysql v1.1.15 // indirect
	code.byted.org/gopkg/circuitbreaker v3.8.0+incompatible // indirect
	code.byted.org/gopkg/consul v1.2.6 // indirect
	code.byted.org/gopkg/ctxvalues v0.5.0 // indirect
	code.byted.org/gopkg/debug v0.10.1 // indirect
	code.byted.org/gopkg/etcd_util v2.3.2+incompatible // indirect
	code.byted.org/gopkg/etcdproxy v0.1.1 // indirect
	code.byted.org/gopkg/logid v0.0.0-20211104042040-f78600e482f2 // indirect
	code.byted.org/gopkg/logs/v2 v2.1.51 // indirect
	code.byted.org/gopkg/metainfo v0.1.4 // indirect
	code.byted.org/gopkg/metrics/v3 v3.1.31 // indirect
	code.byted.org/gopkg/metrics/v4 v4.0.26 // indirect
	code.byted.org/gopkg/metrics_core v0.0.26 // indirect
	code.byted.org/gopkg/mockito v1.3.0 // indirect
	code.byted.org/gopkg/net2 v1.5.0 // indirect
	code.byted.org/gopkg/rand v0.0.0-20200622102840-8cd9b682e5b4 // indirect
	code.byted.org/gopkg/retry v0.0.0-20220517012520-bde92e63db0a // indirect
	code.byted.org/gopkg/stats v1.2.7 // indirect
	code.byted.org/gopkg/thrift v1.6.1 // indirect
	code.byted.org/hystrix/hystrix-go v0.0.0-20190214095017-a2a890c81cd5 // indirect
	code.byted.org/inf/infsecc v1.0.2 // indirect
	code.byted.org/kite/endpoint v3.7.5+incompatible // indirect
	code.byted.org/kite/kitc v3.10.21+incompatible // indirect
	code.byted.org/kite/kite v3.9.30+incompatible // indirect
	code.byted.org/kite/kitutil v3.8.3+incompatible // indirect
	code.byted.org/lang/trace v0.0.2 // indirect
	code.byted.org/lidar/profiler v0.3.2 // indirect
	code.byted.org/lidar/profiler/hertz v0.0.0-20230801111316-7e5562fd8659 // indirect
	code.byted.org/log_market/gosdk v0.0.0-20230524072203-e069d8367314 // indirect
	code.byted.org/log_market/loghelper v0.1.10 // indirect
	code.byted.org/log_market/tracelog v0.1.4 // indirect
	code.byted.org/log_market/ttlogagent_gosdk v0.0.6 // indirect
	code.byted.org/log_market/ttlogagent_gosdk/v4 v4.0.51 // indirect
	code.byted.org/middleware/fic_client v0.2.2 // indirect
	code.byted.org/middleware/gocaller v0.0.4 // indirect
	code.byted.org/security/go-spiffe-v2 v1.0.6 // indirect
	code.byted.org/security/kms-v2-sdk-golang v1.0.9 // indirect
	code.byted.org/security/memfd v0.0.1 // indirect
	code.byted.org/security/scs-go v0.0.5 // indirect
	code.byted.org/security/sensitive_finder_engine v0.3.18 // indirect
	code.byted.org/security/spiffe_spire v0.0.0-20201116193931-c566c1c41bdf // indirect
	code.byted.org/security/zero-trust-identity-helper v1.0.2 // indirect
	code.byted.org/security/zti-jwt-helper-golang v1.0.16 // indirect
	code.byted.org/service_mesh/shmipc v0.2.13 // indirect
	code.byted.org/trace/trace-client-go v1.3.6 // indirect
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible // indirect
	github.com/agiledragon/gomonkey/v2 v2.9.0 // indirect
	github.com/andres-erbsen/clock v0.0.0-20160526145045-9e14626cd129 // indirect
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bytedance/go-tagexpr/v2 v2.9.2 // indirect
	github.com/bytedance/mockey v1.2.9 // indirect
	github.com/bytedance/sonic v1.9.2 // indirect
	github.com/caarlos0/env/v6 v6.10.1 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20221115062448-fe3a3abad311 // indirect
	github.com/choleraehyq/pid v0.0.18 // indirect
	github.com/choleraehyq/rwlock v0.0.13 // indirect
	github.com/cloudwego/netpoll v0.3.2 // indirect
	github.com/deckarep/golang-set v1.8.0 // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/getsentry/sentry-go v0.11.0 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/gin-gonic/gin v1.6.3 // indirect
	github.com/go-jose/go-jose/v3 v3.0.0 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-playground/locales v0.13.0 // indirect
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/go-playground/validator/v10 v10.4.1 // indirect
	github.com/go-sql-driver/mysql v1.7.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/pprof v0.0.0-20221103000818-d260c55eee4c // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hbollon/go-edlib v1.6.0 // indirect
	github.com/henrylee2cn/ameda v1.4.10 // indirect
	github.com/henrylee2cn/goutil v0.0.0-20210127050712-89660552f6f8 // indirect
	github.com/hertz-contrib/http2 v0.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/klauspost/cpuid/v2 v2.2.5 // indirect
	github.com/kuangchanglang/graceful v1.0.2 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/magiconair/properties v1.8.4 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/mitchellh/mapstructure v1.4.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/nyaruka/phonenumbers v1.0.56 // indirect
	github.com/opentracing/opentracing-go v1.2.1-0.20210205174328-3088eee7e4d2 // indirect
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/shirou/gopsutil/v3 v3.22.1 // indirect
	github.com/spf13/afero v1.9.2 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.7.1 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	github.com/tidwall/gjson v1.13.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.2 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	github.com/zeebo/errs v1.3.0 // indirect
	golang.org/x/arch v0.4.0 // indirect
	golang.org/x/crypto v0.23.0 // indirect
	golang.org/x/mod v0.15.0 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230720185612-659f7aaaa771 // indirect
	google.golang.org/grpc v1.56.2 // indirect
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gorm.io/driver/mysql v1.5.1 // indirect
	gorm.io/hints v1.1.2 // indirect
	gorm.io/plugin/dbresolver v1.4.7 // indirect
)
