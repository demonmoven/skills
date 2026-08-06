# 星图IDL规范

<div class="callout">

规范适用于星图内部的thrift idl定义，目前主要指star\_idl代码仓库。作用范围包括该仓库下的所有\*\.thrift文件

本规范将会被大模型学习，最终在star\_idl仓库的CR环节自动给出建议。

**新增时，请用容易理解的自然语言描述清楚规则，最好附上典型的正例和反例**

</div>

#### Struct以及方法的名字定义统一使用驼峰命名法

描述：使用驼峰命名法，而不要使用下划线, 建议首字符大写, 兼容Go

正例：

```Thrift
struct ChargeMmmAccount {  // 此处使用驼峰命名，正确
    1: required i64 account_id
    2: required i64 customer_id
    3: optional AgentInfo agent_info
}
```

反例：

```Thrift
struct charge_mmm_account {   // 此处不应使用下划线风格的命名，错误
    1: required i64 account_id
    2: required i64 customer_id
    3: optional AgentInfo agent_info
}
```

#### Service的名字统一使用驼峰命名法，通常为\{Module\}Service

正例：

```Thrift
service AdStarOrderService {
}
```

反例：

```Thrift
service star_order_server {
}
```

#### 原则上禁止非兼容变更

<div class="callout">

非兼容变更包括：

- 修改已上线字段的id

- 修改已上线字段的类型

- 已有结构体新增或者删除required字段（理论上新增的都是optional），或者修改限定符

- 修改方法名，或service名，或字段名（字段名理论上是可修改的，但前端也引用了idl，通常不建议修改字段名）

</div>

实例1：

```Thrift
// 原结构：
struct PlayStarCollegeCourseReq {
    1: required i64 s_star_id  (api.header="s_star_id")
    2: required i64 course_id
    255: required base.Base Base
}

// 修改结构体，正例：
struct PlayStarCollegeCourseReq {
    1: required i64 s_star_id  (api.header="s_star_id")
    2: required i64 course_id
    3: optional bool filter_param   // 新参数使用optional
    255: required base.Base Base
}

// 修改结构体，反例
struct PlayStarCollegeCourseReq {  
    1: required i64 s_star_id  (api.header="s_star_id")
    2: required i64 course_id
    3: optional bool filter_param   // 新参数使用了required
    255: required base.Base Base
}

```



实例2：

```Thrift
// 原结构：
struct PlayStarCollegeCourseReq {
    1: required i64 s_star_id  (api.header="s_star_id")
    2: required i64 course_id
    255: required base.Base Base
}

// 修改结构体，正例：
struct PlayStarCollegeCourseReq {
    1: required i64 s_star_id  (api.header="s_star_id")
    2: required i64 course_id
    3: optional string course_id_str   // 改数据类型采用新增参数的方式
    255: required base.Base Base
}

// 修改结构体，反例
struct PlayStarCollegeCourseReq {  
    1: required i64 s_star_id  (api.header="s_star_id")
    2: required string course_id       // 直接改现有参数的数据类型，会导致兼容性问题
    255: required base.Base Base
}

```

#### Req和Resp都引用star\_idl统一定义的的base\.thrift，且为required

通常要求我们统一都include公司统一的base\.thrift，但由于历史原因，星图都引用了自定义的star\_idl/idl/base\.thrift，这里我们统一follow现有做法，既不用外部的base\.thrift，也不在每个service维度每次重新定义

> 另外，内部接口通常不对外直接提供，避免与外部冲突
> 
> 

正例：

```Thrift
include "../base.thrift"
struct PlayStarCollegeCourseReq { 
    1: required i64 s_star_id 
    2: required i64 course_id
    255: required base.Base Base   # 直接引用星图统一定义的base.thrift
} 
```

反例：

```Thrift
struct Base {
    1: string LogID = "",
    2: string Caller = "",
    3: string Addr = "",
    4: string Client = "",
    5: optional TrafficEnv TrafficEnv,
    6: optional map<string, string> Extra,
}

struct BaseResp {
    1: string StatusMessage = "",
    2: i32 StatusCode = 0,
    3: optional map<string, string> Extra,
}

struct PlayStarCollegeCourseReq {  
    1: required i64 s_star_id 
    2: required i64 course_id
    255: required base.Base Base    # 在service维度重新定义了一次base，没有必要
}
```

#### Struct的声明必须在使用之前

正例：

```Thrift
struct ResourceWhiteListBo {
    1: required i64 white_list_id
    2: required string expire_time  
    3: required string name
    4: required i64 create_time
}

struct GetAllImcResourceWhiteListResp {
    1: required list<ResourceWhiteListBo> resource_white_list  # 上文先声明ResourceWhiteListBo，再使用
    255: base.BaseResp BaseResp
}
```

反例：

```Thrift

struct GetAllImcResourceWhiteListResp {
    1: required list<ResourceWhiteListBo> resource_white_list  # 未声明ResourceWhiteListBo，先使用
    255: base.BaseResp BaseResp
}

struct ResourceWhiteListBo {
    1: required i64 white_list_id
    2: required string expire_time  
    3: required string name
    4: required i64 create_time
}
```



#### Struct的名字不要使用Result作为后缀，也不要使用New作为前缀

正例：

```Thrift
struct ChargeMmmAccount {  
    1: required i64 account_id
    2: required i64 customer_id
    3: optional AgentInfo agent_info
}
```

反例：

```Thrift
struct NewAccountResult {    # 不要使用New作为前缀，Result作为后缀
    1: required i64 account_id
    2: required i64 customer_id
    3: optional AgentInfo agent_info
}
```

#### 方法只能拥有一个参数，并且这个参数类型必须是自定义的struct类型

描述：参数类型名字使用驼峰命名法，通常为：XXXReq，返回值为XXXResp。XXX一般取方法名，入参和返回值均不和其他方法复用

正例：

```Thrift
struct PlayStarCollegeCourseReq {    # 入参以Req结尾
    1: required i64 s_star_id 
    2: required i64 course_id
    255: required base.Base Base
}
struct PlayStarCollegeCourseResp {   # 返回值以Resp结尾
    255: optional base.BaseResp BaseResp
}
PlayStarCollegeCourseResp PlayStarCollegeCourse(1:PlayStarCollegeCourseReq req)  (api.post="/gw/api/generic/play_star_college_course") 
```

反例：

```Thrift
struct PlayStarCollegeCourseRequest {  # 入参没有以Req结尾
    1: required i64 s_star_id 
    2: required i64 course_id
    255: required base.Base Base
}
struct PlayStarCollegeCourseResponse { # 返回值没有以Resp结尾
    255: optional base.BaseResp BaseResp
}
PlayStarCollegeCourseResponse StarCollegeCourse(1:PlayStarCollegeCourseRequest req)  (api.post="/gw/api/generic/play_star_college_course")  # 方法名与Req和Resp名不对应
```

#### 所有接口都需要在注释中明确是get还是post，如为post需要标明是否带事务

正例：

```Thrift
McnSetBankInfoResp McnSetBankInfo(1: McnSetBankInfoReq req) (api.post="/gw/api/mcn/mcn_set_bank_info")  # http_post use_db_commit mcn填写收款信息    显式标明了http_post
McnSetBankInfoResp McnSetBankInfo(1: McnSetBankInfoReq req) (api.post="/gw/api/mcn/mcn_set_bank_info")  # http_post drop_db_commit mcn填写收款信息    显式标明了http_post
```

反例：

```Thrift
McnSetBankInfoResp McnSetBankInfo(1: McnSetBankInfoReq req) (api.post="/gw/api/mcn/mcn_set_bank_info")  # use_db_commit mcn填写收款信息   没有显式标明http_get还是http_post
McnSetBankInfoResp McnSetBankInfo(1: McnSetBankInfoReq req) (api.post="/gw/api/mcn/mcn_set_bank_info")  # http_post mcn填写收款信息   没有显式标明是否带事务
```

#### Base和BaseResp的规定

每个Req结构必须包含Base字段，且位于255位置，非optional，类型为base\.Base，例如：255: required base\.Base Base

每个Resp结构必须包含BaseResp字段，且位于255位置，非optional，类型为base\.BaseResp，例如：255: base\.BaseResp BaseResp

正例：

```Thrift
struct PlayStarCollegeCourseRequest {  
    1: required i64 s_star_id 
    2: required i64 course_id
    255: required base.Base Base # base字段放在255
}
```

反例：

```Thrift
struct PlayStarCollegeCourseRequest {  
    1: optional base.Base Base_1 # base字段下标不对，命名不对，另外应该是required
    2: required i64 s_star_id 
    3: required i64 course_id
}
```

#### 结构体的字段采用下划线的命名风格，新增字段必须有限定符required/optional

正例：

```Thrift
struct PlayStarCollegeCourseRequest {  
    1: required i64 s_star_id 
    2: required i64 course_id 
    3: optional bool need_extra  # 新增字段必须加限定符
    255: required base.Base Base 
}
```

反例：

```Thrift
struct PlayStarCollegeCourseRequest {  
    1: required i64 s_star_id 
    2: required i64 course_id 
    3: bool needExtra   # 新增字段未加限定符，命名风格错误
    255: required base.Base Base 
}
```

#### 命名为s\_\*\_id的登录态相关参数，必须加tag \(api\.header\)

正例：

```Thrift
struct GetHotListTabV2Req{
    1: required i64 s_star_id (api.header = 's_star_id')  # 登录态参数加tag，避免被参数覆盖
    2: required string category
    255: required base.Base Base
}
```

反例：

```Thrift
struct GetHotListTabV2Req{
    1: required i64 s_star_id  # 登录态参数未加tag
    2: required string category
    255: required base.Base Base
}
```

#### 内部调用的rpc接口用Inner开头，不需要登录态参数，不加agw配置

描述：内部调用接口没有登录态校验和严格的水平鉴权，不能直接对外放开

正例：

```Thrift
struct InnerClearAllUnreadReq{
    1: required i64 star_id,
    255: required base.Base Base
}

struct InnerClearAllUnreadResp{
    255: required base.BaseResp BaseResp
}
InnerClearAllUnreadResp InnerClearAllUnread(1: InnerClearAllUnreadReq req) # http_post use_db_commit 清除所有未读
```

反例：

```Thrift
struct InnerClearAllUnreadReq{
    1: required i64 star_id,
    255: required base.Base Base
}

struct InnerClearAllUnreadResp{
    255: required base.BaseResp BaseResp
}
InnerClearAllUnreadResp InnerClearAllUnread(1: InnerClearAllUnreadReq req) (api.post="/gw/api/generic/inner_clear_all_unread") # http_post use_db_commit 清除所有未读  此处不应该加上agw配置
```

#### 直接暴露给前端的接口，不使用Inner开头，非特殊情况必须加登录态

正例：

```Thrift
struct GetAutoReplyReq{
    1: required i64 s_star_id (api.header="s_star_id")  // 必须有登录态
    2: optional i64 author_id
    255: required base.Base Base
}
struct GetAutoReplyResp{
    1: required string reply_text
    2: required DemanderType demander_type
    3: required TriggerCondition trigger_condition
    255: required base.BaseResp BaseResp
}
GetAutoReplyResp GetAutoReply(1: GetAutoReplyReq req)  (api.get="/gw/api/generic/get_auto_reply") # http_get 获取自动回复
```

反例：

```Thrift
struct GetAutoReplyReq{
    1: required i64 star_id   // 没有登录态注入，可能导致接口被滥用
    2: optional i64 author_id
    255: required base.Base Base
}
struct GetAutoReplyResp{
    1: required string reply_text
    2: required DemanderType demander_type
    3: required TriggerCondition trigger_condition
    255: required base.BaseResp BaseResp
}
GetAutoReplyResp GetAutoReply(1: GetAutoReplyReq req)  (api.get="/gw/api/generic/get_auto_reply") # http_get 获取自动回复
```

#### 结构体参数须规避特定的关键字

原因可能来自：与thrift\_gen中的方法重名（如read），与Chrome自带的header名字重名（如priority）等

不能作为字段名的关键字列表：

```Thrift
['device_type', 'need_personal_recommend', 'version_code', 'package', 'js_sdk_version', 'tma_jssdk_version', 'foo', 'app_name', 'app_version', 'device_id', 'channel', 'mcc_mnc', 'aid', 'minor_status', 'screen_width', 'klink_egdi', 'cdid', 'os_api', 'ac', 'os_version', 'appTheme', 'is_guest_mode', 'device_platform', 'build_number', 'iid', 'is_vcd', 'request_tag_from', 'version_name', 'os', 'sub_os_api', 'ssmix', 'device_brand', 'language', 'manifest_version_code', 'resolution', 'dpi', 'update_version_code', '_rticket', 'first_launch_timestamp', 'last_deeplink_update_version_code', 'cpu_support64', 'host_abi', 'app_type', 'is_preinstall', 'is_android_pad', 'is_android_fold', 'ts', 'read', 'priority']
```

正例：

```Thrift
struct GetAutoReplyReq{
    1: required i64 star_id 
    2: optional i64 author_id  
    3: optional i64 read_priority  # 未命中关键字
    255: required base.Base Base
}
```

反例：

```Thrift
struct GetAutoReplyReq{
    1: required i64 star_id 
    2: optional i64 author_id  
    3: optional i64 priority  # 未命了priority关键字，可能导致获取不到正确的值
    255: required base.Base Base
}
```

#### IDL中新增的数据模型struct和枚举类型，不要和服务接口定义写在同一文件里。否则当其他文件对其进行引用时，可能会出现重复定义或者循环引用问题

有专门用于存放枚举的consts目录和数据模型model目录

#### 禁止使用i8类型，如果存在需要使用的情况用i16替代
