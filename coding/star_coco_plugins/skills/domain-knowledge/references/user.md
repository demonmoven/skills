# 星图账号领域开发者手册

<table><tbody>
<tr>
<td>

版本

</td>
<td>

更新日期

</td>
<td>

更新人

</td>
<td>

说明

</td>
</tr>
<tr>
<td>

v0\.2

</td>
<td>

2023\-04\-07

</td>
<td>

@陈明

</td>
<td>

细化各个接口描述

</td>
</tr>
<tr>
<td>

v0\.1

</td>
<td>

2025\-04\-01

</td>
<td>

@陈明

</td>
<td>

初稿

</td>
</tr>
</tbody></table>

> 文档定位：面向星图内其他领域的后端RD、星图外/集团内二方的RD，在与本领域有逻辑交互时的开发文档
> 
> 维护方式：更新后在文档开头记录版本，并更新到飞书系统中
> 
> 

## 1\.领域概述

> 简述本领域cover的范围、核心概念和业务流程。仅做开发所必需的知识科普，避免长篇大论
> 
> 视实际情况补充，也可以按需补充在接口文档里
> 
> 

### 1\.1核心实体/聚合根

> 如：结算单、结算流水
> 
> 

#### 账号：

##### 平台来源

```Thrift
enum PlatformSource {
    douyin = 1          # 抖音
    toutiao = 2         # 头条
    xigua = 3           # 西瓜
~~    huoshan = 4         # 火山 ~~~~已下线~~
~~    creator = 5         # 创作者~~~~ 已下线~~
    ocean_engine = 6    # 巨量引擎
}
```

##### **BasicCoreUserInfo 基础信息**

<table><tbody>
<tr>
<td>

字段

</td>
<td>

类型

</td>
<td>

必选

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

id

</td>
<td>

i64

</td>
<td>

required

</td>
<td>

用户唯一ID

</td>
</tr>
<tr>
<td>

nick\_name

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

昵称

</td>
</tr>
<tr>
<td>

email

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

邮箱

</td>
</tr>
<tr>
<td>

phone

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

绑定手机号

</td>
</tr>
<tr>
<td>

status

</td>
<td>

CoreUserStatus

</td>
<td>

optional

</td>
<td>

账号状态

</td>
</tr>
<tr>
<td>

create\_time

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

创建时间戳

</td>
</tr>
<tr>
<td>

avatar\_url

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

头像地址

</td>
</tr>
<tr>
<td>

plain\_phone

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

明文电话

</td>
</tr>
<tr>
<td>

plain\_email

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

明文邮箱

</td>
</tr>
<tr>
<td>

mobile\_id

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

手机号ID

</td>
</tr>
<tr>
<td>

province

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

省份

</td>
</tr>
<tr>
<td>

city

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

城市

</td>
</tr>
<tr>
<td>

follower\_cnt

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

粉丝数

</td>
</tr>
<tr>
<td>

gender

</td>
<td>

i16

</td>
<td>

optional

</td>
<td>

性别

</td>
</tr>
</tbody></table>

##### **DouYinUserInfo 抖音信息**

<table><tbody>
<tr>
<td>

字段

</td>
<td>

类型

</td>
<td>

必选

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

is\_blue\_v

</td>
<td>

bool

</td>
<td>

optional

</td>
<td>

认证抖音蓝V

</td>
</tr>
<tr>
<td>

role\_list

</td>
<td>

list\&lt;i64\&gt;

</td>
<td>

optional

</td>
<td>

角色ID列表

</td>
</tr>
<tr>
<td>

unique\_id

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

抖音号

</td>
</tr>
<tr>
<td>

short\_id

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

短ID

</td>
</tr>
<tr>
<td>

valid\_follwer\_cnt

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

有效粉丝数量

</td>
</tr>
<tr>
<td>

highest\_valid\_follwer\_cnt

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

历史最高有效粉丝数量

</td>
</tr>
<tr>
<td>

is\_ecom

</td>
<td>

bool

</td>
<td>

optional

</td>
<td>

是否是电商

</td>
</tr>
</tbody></table>

##### **TouTiaoUserInfo 头条信息**

<table><tbody>
<tr>
<td>

字段

</td>
<td>

类型

</td>
<td>

必选

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

mid

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

头条号mid

</td>
</tr>
</tbody></table>

##### **XiGuaUserInfo 西瓜信息**

<table><tbody>
<tr>
<td>

字段

</td>
<td>

类型

</td>
<td>

必选

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

mid

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

头条号mid

</td>
</tr>
</tbody></table>

##### Oauth 三方授权

<table><tbody>
<tr>
<td>

关联账号

</td>
<td>

关联账号平台来源

</td>
<td>

被关联账号

</td>
<td>

关联账号平台来源

</td>
</tr>
<tr>
<td>

i64

</td>
<td>

platform\_source

</td>
<td>

i64

</td>
<td>

platform\_source

</td>
</tr>
</tbody></table>

#### 账户：

账户类型

人户关系

角色权限



信息

##### **BasicAccountInfo 基础信息**

<table><tbody>
<tr>
<td>

字段

</td>
<td>

类型

</td>
<td>

必选

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

id

</td>
<td>

i64

</td>
<td>

required

</td>
<td>

账户唯一ID

</td>
</tr>
<tr>
<td>

nick\_name

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

昵称

</td>
</tr>
<tr>
<td>

role

</td>
<td>

i64

</td>
<td>

required

</td>
<td>

角色

</td>
</tr>
<tr>
<td>

origin

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

注册来源

</td>
</tr>
<tr>
<td>

status

</td>
<td>

AccountStatus

</td>
<td>

optional

</td>
<td>

账户状态

</td>
</tr>
<tr>
<td>

create\_time

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

创建时间戳

</td>
</tr>
<tr>
<td>

qualification\_info

</td>
<td>

QualificationInfo

</td>
<td>

optional

</td>
<td>

资质信息

</td>
</tr>
<tr>
<td>

customer\_info

</td>
<td>

CustomerInfo

</td>
<td>

optional

</td>
<td>

客销

</td>
</tr>
</tbody></table>

**DemanderInfo 客户信息**

<table><tbody>
<tr>
<td>

字段

</td>
<td>

类型

</td>
<td>

必选

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

grade

</td>
<td>

i64

</td>
<td>

required

</td>
<td>

等级

</td>
</tr>
<tr>
<td>

brands

</td>
<td>

list\&lt;BrandInfo\&gt;

</td>
<td>

optional

</td>
<td>

品牌

</td>
</tr>
</tbody></table>

**AuthorInfo 达人信息**

<table><tbody>
<tr>
<td>

字段

</td>
<td>

类型

</td>
<td>

必选

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

identity\_code

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

身份证号

</td>
</tr>
<tr>
<td>

identity\_name

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

身份证名

</td>
</tr>
<tr>
<td>

withdraw\_phone

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

提现手机号

</td>
</tr>
</tbody></table>

**ProviderInfo 服务商信息**

<table><tbody>
<tr>
<td>

字段

</td>
<td>

类型

</td>
<td>

必选

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

service\_type

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

服务商功能权限

</td>
</tr>
<tr>
<td>

grade

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

服务商等级

</td>
</tr>
<tr>
<td>

evaluation

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

评分

</td>
</tr>
</tbody></table>

**McnInfo 服务商信息**

<table><tbody>
<tr>
<td>

字段

</td>
<td>

类型

</td>
<td>

必选

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

douyin\_mcn\_id

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

抖音mcn\_id

</td>
</tr>
</tbody></table>

**LifeAccountInfo 来客信息**

<table><tbody>
<tr>
<td>

字段

</td>
<td>

类型

</td>
<td>

必选

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

account\_id

</td>
<td>

i64

</td>
<td>

required

</td>
<td>

来客账户id

</td>
</tr>
<tr>
<td>

poi

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

店铺poi信息

</td>
</tr>
</tbody></table>

**LifeAccountInfo 来客信息**

<table><tbody>
<tr>
<td>

字段

</td>
<td>

类型

</td>
<td>

必选

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

account\_id

</td>
<td>

i64

</td>
<td>

required

</td>
<td>

来客账户id

</td>
</tr>
<tr>
<td>

poi

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

店铺poi信息

</td>
</tr>
</tbody></table>

**SmallGameAccountInfo 小游戏客户信息**

<table><tbody>
<tr>
<td>

字段

</td>
<td>

类型

</td>
<td>

必选

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

app\_id

</td>
<td>

i64

</td>
<td>

required

</td>
<td>

小游戏应用id

</td>
</tr>
</tbody></table>

**OpenMerchantAccountInfo 小程序客户信息**

<table><tbody>
<tr>
<td>

字段

</td>
<td>

类型

</td>
<td>

必选

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

app\_id

</td>
<td>

i64

</td>
<td>

required

</td>
<td>

小游戏应用id

</td>
</tr>
<tr>
<td>

merchant\_id

</td>
<td>

i64

</td>
<td>

required

</td>
<td>

商户号

</td>
</tr>
</tbody></table>



#### 资质：[巨量钥匙认证中心产品介绍](https://bytedance.larkoffice.com/wiki/wikcne1kAay2SmLYvRx5v5r19y4)

**资质状态 \&amp; 主体类型**

```json
enum AccountStatus {
    not_start     = 0
    waiting       = 1
    processing    = 2
    success       = 3
    failed        = 4
    expired       = 5
}
enum SubjectType {
    company = 1 # 企业
    person = 2 # 个人
}
```

**QualificationInfo 资质信息**

<table><tbody>
<tr>
<td>

字段

</td>
<td>

类型

</td>
<td>

必选

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

account\_id

</td>
<td>

i64

</td>
<td>

required

</td>
<td>

账户唯一ID

</td>
</tr>
<tr>
<td>

qualification\_status

</td>
<td>

AccountStatus

</td>
<td>

optional

</td>
<td>

资质总状态

</td>
</tr>
<tr>
<td>

verification\_status

</td>
<td>

AccountStatus

</td>
<td>

optional

</td>
<td>

认证状态

</td>
</tr>
<tr>
<td>

ca\_status

</td>
<td>

AccountStatus

</td>
<td>

optional

</td>
<td>

签章状态

</td>
</tr>
<tr>
<td>

crm\_status

</td>
<td>

AccountStatus

</td>
<td>

optional

</td>
<td>

客销状态

</td>
</tr>
<tr>
<td>

subject\_type

</td>
<td>

SubjectType

</td>
<td>

optional

</td>
<td>

主体类型

</td>
</tr>
<tr>
<td>

qualification\_industry\_status

</td>
<td>

AccountStatus

</td>
<td>

optional

</td>
<td>

行业\+主体资质审核状态

</td>
</tr>
</tbody></table>

**IdentityInfo 实名信息**

```json
/**
**第二代居民身份证**：编号为1。
**定居国外的中国公民护照**：编号为2。
**港澳居民来往内地通行证**：编号为3。
**台湾居民来往大陆通行证**：编号为4。
**外国人永久居留身份证**：编号为5。
**台湾居民居住证**：编号为6。
**外国护照**：编号为7 。
**港澳居民居住证**：编号为8。
**其他证件类型**：编号为99 。
*/
type IdentityType int64
```

<table><tbody>
<tr>
<td>

字段

</td>
<td>

类型

</td>
<td>

必选

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

user\_id

</td>
<td>

i64

</td>
<td>

required

</td>
<td>

唯一ID\(一般是star\_id\)

</td>
</tr>
<tr>
<td>

identity\_name

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

姓名

</td>
</tr>
<tr>
<td>

identity\_code

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

身份证号

</td>
</tr>
<tr>
<td>

identity\_type

</td>
<td>

IdentityType

</td>
<td>

optional

</td>
<td>

证件类型

</td>
</tr>
<tr>
<td>

is\_adult

</td>
<td>

bool

</td>
<td>

optional

</td>
<td>

是否成年

</td>
</tr>
<tr>
<td>

withdraw\_phone

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

提现手机号

</td>
</tr>
</tbody></table>

**CompanyInfo 公司信息**

<table><tbody>
<tr>
<td>

字段

</td>
<td>

类型

</td>
<td>

必选

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

id

</td>
<td>

i64

</td>
<td>

required

</td>
<td>

公司id

</td>
</tr>
<tr>
<td>

name

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

公司名

</td>
</tr>
<tr>
<td>

qualification\_serial

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

主体资质编号

</td>
</tr>
<tr>
<td>

subject\_type

</td>
<td>

SubjectType

</td>
<td>

optional

</td>
<td>

主体类型

</td>
</tr>
<tr>
<td>

proprietor\_name

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

法人

</td>
</tr>
</tbody></table>

#### 客销：

```json
enum CustomerCategory {
    direct_cus = 1
    agent_cus = 2
    virtual_cus = 4
    parnter_cus = 6
    self_cus = 7
    vip_cus = 8
}
```

<table><tbody>
<tr>
<td>

字段

</td>
<td>

类型

</td>
<td>

必选

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

customer\_id

</td>
<td>

i64

</td>
<td>

required

</td>
<td>

客户id

</td>
</tr>
<tr>
<td>

customer\_name

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

客户名

</td>
</tr>
<tr>
<td>

type

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

客户类型

</td>
</tr>
<tr>
<td>

category\_id

</td>
<td>

CustomerCategory

</td>
<td>

optional

</td>
<td>

客户分类

</td>
</tr>
<tr>
<td>

is\_virtual

</td>
<td>

bool

</td>
<td>

optional

</td>
<td>

是否虚客

</td>
</tr>
<tr>
<td>

is\_agent

</td>
<td>

bool

</td>
<td>

optional

</td>
<td>

是否代理

</td>
</tr>
<tr>
<td>

subject\_type

</td>
<td>

SubjectType

</td>
<td>

optional

</td>
<td>

主体类型

</td>
</tr>
<tr>
<td>

sale\_id

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

销售员工id

</td>
</tr>
<tr>
<td>

sale\_department\_id

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

销售员工部门号

</td>
</tr>
<tr>
<td>

sale\_department\_name

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

销售员工部门名

</td>
</tr>
</tbody></table>

#### 合同：

```json
enum StampStatus {
    unstamped = 1
    stamping = 2
    stamped = 3
}

enum DepositStatus {
    not_complete = 1
    completed = 3
}
```

**ContractInfo 合同信息**

<table><tbody>
<tr>
<td>

字段

</td>
<td>

类型

</td>
<td>

必选

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

id

</td>
<td>

i64

</td>
<td>

required

</td>
<td>

合同id

</td>
</tr>
<tr>
<td>

cont\_name

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

合同名

</td>
</tr>
<tr>
<td>

contract\_type

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

合同类型

</td>
</tr>
<tr>
<td>

customer\_id

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

客户id

</td>
</tr>
<tr>
<td>

stamp\_status

</td>
<td>

StampStatus

</td>
<td>

optional

</td>
<td>

签章状态

</td>
</tr>
<tr>
<td>

ca\_id

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

签章id

</td>
</tr>
<tr>
<td>

deposit\_status

</td>
<td>

DepositStatus

</td>
<td>

optional

</td>
<td>

保证金状态

</td>
</tr>
<tr>
<td>

deposit\_url

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

保证金查询地址

</td>
</tr>
<tr>
<td>

sign\_url

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

签章地址

</td>
</tr>
<tr>
<td>

subject\_id

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

我方主体id

</td>
</tr>
<tr>
<td>

subject\_name

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

我方主体名称

</td>
</tr>
<tr>
<td>

start\_time

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

合同生效时间

</td>
</tr>
<tr>
<td>

end\_time

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

合同过期时间

</td>
</tr>
</tbody></table>

**签章：**

<table><tbody>
<tr>
<td>

字段

</td>
<td>

类型

</td>
<td>

必选

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

id

</td>
<td>

i64

</td>
<td>

required

</td>
<td>

合同id

</td>
</tr>
<tr>
<td>

company\_name

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

公司名

</td>
</tr>
<tr>
<td>

qualification\_type

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

资质类型

</td>
</tr>
<tr>
<td>

qualification\_serial

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

主体资质编号

</td>
</tr>
<tr>
<td>

fdd\_customer\_id

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

法大大客户id

</td>
</tr>
<tr>
<td>

application\_type

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

申请方类型

</td>
</tr>
<tr>
<td>

applicant\_email

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

申请方邮箱

</td>
</tr>
<tr>
<td>

applicant\_name

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

申请方名称

</td>
</tr>
<tr>
<td>

applicant\_mobile

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

申请方手机

</td>
</tr>
</tbody></table>

#### 协议：

<table><tbody>
<tr>
<td>

字段

</td>
<td>

类型

</td>
<td>

必选

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

id

</td>
<td>

i64

</td>
<td>

required

</td>
<td>

协议id

</td>
</tr>
<tr>
<td>

version

</td>
<td>

i64

</td>
<td>

optional

</td>
<td>

协议版本

</td>
</tr>
<tr>
<td>

name

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

协议名称

</td>
</tr>
<tr>
<td>

content

</td>
<td>

string

</td>
<td>

optional

</td>
<td>

协议内容

</td>
</tr>
</tbody></table>

### 1\.2通用概念/术语

> 如：计费单、普通订单/父子单
> 
> 

## 2\.API调用指南

### 账号

#### 信息查询

<table><tbody>
<tr>
<td>

接口名称

</td>
<td>

InnerGetCoreUserInfoV2

</td>
<td>

psm

</td>
<td>

ad\.star\.gouser

</td>
</tr>
<tr>
<td>

维护人

</td>
<td>

@何家为

</td>
<td>

能承受的qps

</td>
<td>



</td>
</tr>
<tr>
<td>

接口定义

\(推荐使用BAM接口连接，也可以手工维护idl\)



</td>
<td colspan="3">



</td>
</tr>
<tr>
<td>

调用依赖



</td>
<td colspan="3">

```Plain Text
Go:  go get code.byted.org/overpass/ad_star_gouser@latest
Python: from star_common.gateway.euler_base_rpc import go_user_client
```

</td>
</tr>
<tr>
<td>

注意事项



</td>
<td colspan="3">



</td>
</tr>
</tbody></table>



1. 账号ID互查

    1. 外部关联账号 \&lt;\-\-\-\&gt; 星图商业化UID

<table><tbody>
<tr>
<td>

接口名称

</td>
<td>

InnerGetStarIdByOwnerUid

</td>
<td>

psm

</td>
<td>

ad\.star\.gouser

</td>
</tr>
<tr>
<td>

维护人

</td>
<td>

@何家为

</td>
<td>

能承受的qps

</td>
<td>



</td>
</tr>
<tr>
<td>

接口定义

\(推荐使用BAM接口连接，也可以手工维护idl\)



</td>
<td colspan="3">



</td>
</tr>
<tr>
<td>

调用依赖



</td>
<td colspan="3">

```Plain Text
Go:  go get code.byted.org/overpass/ad_star_gouser@latest
Python: from star_common.gateway.euler_base_rpc import go_user_client
```

</td>
</tr>
<tr>
<td>

注意事项



</td>
<td colspan="3">



</td>
</tr>
</tbody></table>



<table><tbody>
<tr>
<td>

接口名称

</td>
<td>

InnerGetOwnerUidByStarId

</td>
<td>

psm

</td>
<td>

ad\.star\.gouser

</td>
</tr>
<tr>
<td>

维护人

</td>
<td>

@何家为

</td>
<td>

能承受的qps

</td>
<td>



</td>
</tr>
<tr>
<td>

接口定义

\(推荐使用BAM接口连接，也可以手工维护idl\)



</td>
<td colspan="3">



</td>
</tr>
<tr>
<td>

调用依赖



</td>
<td colspan="3">

```Plain Text
Go:  go get code.byted.org/overpass/ad_star_gouser@latest
Python: from star_common.gateway.euler_base_rpc import go_user_client
```

</td>
</tr>
<tr>
<td>

注意事项



</td>
<td colspan="3">



</td>
</tr>
</tbody></table>



<table><tbody>
<tr>
<td>

接口名称

</td>
<td>

InnerGetCoreUserIdByStarIdV2

</td>
<td>

psm

</td>
<td>

ad\.star\.gouser

</td>
</tr>
<tr>
<td>

维护人

</td>
<td>

@何家为

</td>
<td>

能承受的qps

</td>
<td>



</td>
</tr>
<tr>
<td>

接口定义

\(推荐使用BAM接口连接，也可以手工维护idl\)



</td>
<td colspan="3">



</td>
</tr>
<tr>
<td>

调用依赖



</td>
<td colspan="3">

```Plain Text
Go:  go get code.byted.org/overpass/ad_star_gouser@latest
Python: from star_common.gateway.euler_base_rpc import go_user_client
```

</td>
</tr>
<tr>
<td>

注意事项



</td>
<td colspan="3">



</td>
</tr>
</tbody></table>



<table><tbody>
<tr>
<td>

接口名称

</td>
<td>

InnerGetUserIdByCoreUserId

</td>
<td>

psm

</td>
<td>

ad\.star\.gouser

</td>
</tr>
<tr>
<td>

维护人

</td>
<td>

@何家为

</td>
<td>

能承受的qps

</td>
<td>



</td>
</tr>
<tr>
<td>

接口定义

\(推荐使用BAM接口连接，也可以手工维护idl\)



</td>
<td colspan="3">



</td>
</tr>
<tr>
<td>

调用依赖



</td>
<td colspan="3">

```Plain Text
Go:  go get code.byted.org/overpass/ad_star_gouser@latest
Python: from star_common.gateway.euler_base_rpc import go_user_client
```

</td>
</tr>
<tr>
<td>

注意事项



</td>
<td colspan="3">



</td>
</tr>
</tbody></table>

<table><tbody>
<tr>
<td>

接口名称

</td>
<td>

InnerGetCoreUserIdByStarIdV2

</td>
<td>

psm

</td>
<td>

ad\.star\.gouser

</td>
</tr>
<tr>
<td>

维护人

</td>
<td>

@何家为

</td>
<td>

能承受的qps

</td>
<td>



</td>
</tr>
<tr>
<td>

接口定义

\(推荐使用BAM接口连接，也可以手工维护idl\)



</td>
<td colspan="3">



</td>
</tr>
<tr>
<td>

调用依赖



</td>
<td colspan="3">

```Plain Text
Go:  go get code.byted.org/overpass/ad_star_gouser@latest
Python: from star_common.gateway.euler_base_rpc import go_user_client
```

</td>
</tr>
<tr>
<td>

注意事项



</td>
<td colspan="3">



</td>
</tr>
</tbody></table>

<table><tbody>
<tr>
<td>

接口名称

</td>
<td>

InnerGetMediaUidByOcUid

</td>
<td>

psm

</td>
<td>

ad\.star\.gouser

</td>
</tr>
<tr>
<td>

维护人

</td>
<td>

@何家为

</td>
<td>

能承受的qps

</td>
<td>



</td>
</tr>
<tr>
<td>

接口定义

\(推荐使用BAM接口连接，也可以手工维护idl\)



</td>
<td colspan="3">



</td>
</tr>
<tr>
<td>

调用依赖



</td>
<td colspan="3">

```Plain Text
Go:  go get code.byted.org/overpass/ad_star_gouser@latest
Python: from star_common.gateway.euler_base_rpc import go_user_client
```

</td>
</tr>
<tr>
<td>

注意事项



</td>
<td colspan="3">



</td>
</tr>
</tbody></table>

<table><tbody>
<tr>
<td>

接口名称

</td>
<td>

InnerGetOcUidByMediaUid

</td>
<td>

psm

</td>
<td>

ad\.star\.gouser

</td>
</tr>
<tr>
<td>

维护人

</td>
<td>

@何家为

</td>
<td>

能承受的qps

</td>
<td>



</td>
</tr>
<tr>
<td>

接口定义

\(推荐使用BAM接口连接，也可以手工维护idl\)



</td>
<td colspan="3">



</td>
</tr>
<tr>
<td>

调用依赖



</td>
<td colspan="3">

```Plain Text
Go:  go get code.byted.org/overpass/ad_star_gouser@latest
Python: from star_common.gateway.euler_base_rpc import go_user_client
```

</td>
</tr>
<tr>
<td>

注意事项



</td>
<td colspan="3">



</td>
</tr>
</tbody></table>

### 账户

1. 账户基础信息查询

    1. 星图角色基础信息查询

        1. 账户角色

        2. 账户角色业务信息\(等级，评分，权限\)

    2. 外部账户业务信息查询

2. 账号\-账户关系映射

3. 账户\-账户关系映射



### 资质

1. 账户资质状态查询

2. 达人实名状态查询

3. 账户所属公司信息查询

<table><tbody>
<tr>
<td>

接口名称

</td>
<td>

InnerGetQualificationStatus

</td>
<td>

psm

</td>
<td>

ad\.star\.gouser

</td>
</tr>
<tr>
<td>

维护人

</td>
<td>

@何家为

</td>
<td>

能承受的qps

</td>
<td>



</td>
</tr>
<tr>
<td>

接口定义

\(推荐使用BAM接口连接，也可以手工维护idl\)



</td>
<td colspan="3">



</td>
</tr>
<tr>
<td>

调用依赖



</td>
<td colspan="3">

```Plain Text
Go:  go get code.byted.org/overpass/ad_star_gouser@latest
Python: from star_common.gateway.euler_base_rpc import go_user_client
```

</td>
</tr>
<tr>
<td>

注意事项



</td>
<td colspan="3">



</td>
</tr>
</tbody></table>



### 客销

1. 账户客户查询

2. 账户代理信息查询

3. 销售和其部门信息查询

<table><tbody>
<tr>
<td>

接口名称

</td>
<td>

InnerGetCustomerInfoByStarId

</td>
<td>

psm

</td>
<td>

ad\.star\.gouser

</td>
</tr>
<tr>
<td>

维护人

</td>
<td>

@何家为

</td>
<td>

能承受的qps

</td>
<td>



</td>
</tr>
<tr>
<td>

接口定义

\(推荐使用BAM接口连接，也可以手工维护idl\)



</td>
<td colspan="3">



</td>
</tr>
<tr>
<td>

调用依赖



</td>
<td colspan="3">

```Plain Text
Go:  go get code.byted.org/overpass/ad_star_gouser@latest
Python: from star_common.gateway.euler_base_rpc import go_user_client
```

</td>
</tr>
<tr>
<td>

注意事项



</td>
<td colspan="3">



</td>
</tr>
</tbody></table>

### 合同

1. 合同信息查询

2. 按场景查询可用合同\(星广等复杂场景\)

3. 合同签约

4. 



<table><tbody>
<tr>
<td>

接口名称

</td>
<td>

InnerGetContractInfo

</td>
<td>

psm

</td>
<td>

ad\.star\.gouser

</td>
</tr>
<tr>
<td>

维护人

</td>
<td>

@何家为

</td>
<td>

能承受的qps

</td>
<td>



</td>
</tr>
<tr>
<td>

接口定义

\(推荐使用BAM接口连接，也可以手工维护idl\)



</td>
<td colspan="3">



</td>
</tr>
<tr>
<td>

调用依赖



</td>
<td colspan="3">

```Plain Text
Go:  go get code.byted.org/overpass/ad_star_gouser@latest
Python: from star_common.gateway.euler_base_rpc import go_user_client
```

</td>
</tr>
<tr>
<td>

注意事项



</td>
<td colspan="3">



</td>
</tr>
</tbody></table>

## 3\.事件

- 账户创建\(客户, 机构, 服务商, 达人\)

<table><tbody>
<tr>
<td>

事件名称

</td>
<td>

账户创建

**star\.user\.account\.create**

</td>
<td>

事件类型

</td>
<td>

eventbus

</td>
</tr>
<tr>
<td>

事件描述

</td>
<td colspan="3">

订阅星图角色账户创建\(客户, 机构, 服务商, 达人\)消息

</td>
</tr>
<tr>
<td>

链接

</td>
<td colspan="3">



</td>
</tr>
<tr>
<td>

维护人

</td>
<td>

@何家为

</td>
<td>

qps

</td>
<td>

50

</td>
</tr>
<tr>
<td>

消息结构

与示例



</td>
<td colspan="3">

```Plain Text
{
  "star_id": 1639752636991496,
  "core_user_id": 94245903965,
  "role": 1,
  "origin": 1,
  "event_time": 1744636593
}
```

</td>
</tr>
</tbody></table>

- 账户注销

<table><tbody>
<tr>
<td>

事件名称

</td>
<td>

账户注销

**star\.user\.account\.logoff**

</td>
<td>

事件类型

</td>
<td>

eventbus

</td>
</tr>
<tr>
<td>

事件描述

</td>
<td colspan="3">

订阅星图角色账户注销\(客户, 机构, 服务商, 达人\)消息

</td>
</tr>
<tr>
<td>

链接

</td>
<td colspan="3">



</td>
</tr>
<tr>
<td>

维护人

</td>
<td>

@何家为

</td>
<td>

qps

</td>
<td>

50

</td>
</tr>
<tr>
<td>

消息结构

与示例



</td>
<td colspan="3">

```Plain Text
{
  "star_id": 1639752636991496,
  "core_user_id": 94245903965,
  "role": 1,
  "event_time": 1744636593
}
```

</td>
</tr>
</tbody></table>

- 协议签署

- 合同创建

<table><tbody>
<tr>
<td>

事件名称

</td>
<td>

合同创建

**star\.user\.contract\.create**

</td>
<td>

事件类型

</td>
<td>

eventbus

</td>
</tr>
<tr>
<td>

事件描述

</td>
<td colspan="3">

订阅星图账户合同创建消息

</td>
</tr>
<tr>
<td>

链接

</td>
<td colspan="3">



</td>
</tr>
<tr>
<td>

维护人

</td>
<td>

@何家为

</td>
<td>

qps

</td>
<td>

50

</td>
</tr>
<tr>
<td>

消息结构

与示例



</td>
<td colspan="3">

```Plain Text
{
  "star_id": 1,
  "contract_id": 1,
  "contract_type": 1,
  "subject_id": 1,
  "effect_time": 1744636593,
  "event_time": 1744636593
}
```

</td>
</tr>
</tbody></table>

- 关系变更

    - 人户

    - 媒体关系

    - 垂类

    - 达人\&amp;机构

    - 达人\&amp;服务商

<table><tbody>
<tr>
<td>

事件名称

</td>
<td>

关系变更

**star\.user\.relation\.update**

</td>
<td>

事件类型

</td>
<td>

eventbus

</td>
</tr>
<tr>
<td>

事件描述

</td>
<td colspan="3">

订阅星图账户关系变更消息

</td>
</tr>
<tr>
<td>

链接

</td>
<td colspan="3">



</td>
</tr>
<tr>
<td>

维护人

</td>
<td>

@何家为

</td>
<td>

qps

</td>
<td>

50

</td>
</tr>
<tr>
<td>

消息结构

与示例



</td>
<td colspan="3">

```Plain Text
{
  "relation_type": 1, # 关系类型
  "operate_type": 1, # 1 绑定 / 2 解绑
  "main_id": 1, # 主账户id
  "bind_id": 1, # 绑定关联的账户id
  "event_time": 1744636593
}
```

</td>
</tr>
</tbody></table>

- 信息变更

    - 基础信息

    - 认证信息

        - 实名信息

        - 对公信息

    - 财务信息

        - 银行卡\(对公对私\)

        - 支付宝

<table><tbody>
<tr>
<td>

事件名称

</td>
<td>

关系变更

**star\.user\.info\.update**

</td>
<td>

事件类型

</td>
<td>

eventbus

</td>
</tr>
<tr>
<td>

事件描述

</td>
<td colspan="3">

订阅星图账户信息变更消息

</td>
</tr>
<tr>
<td>

链接

</td>
<td colspan="3">



</td>
</tr>
<tr>
<td>

维护人

</td>
<td>

@何家为

</td>
<td>

qps

</td>
<td>

50

</td>
</tr>
<tr>
<td>

消息结构

与示例



</td>
<td colspan="3">

```Plain Text
{
  "info_type": 1, # 信息类型
  "operate_type": 1, # 1 绑定 / 2 解绑
  "user_id": "123456",
  "changed_fields": [{
      "field": "name",
      "old_value": "张三",
      "new_value": "张四"
   },{
       "field": "mobile",
       "old_value": "13800138000",
       "new_value": "13900139000"
    }, {
        "field": "card_number",
        "old_value": "621785******1234",
        "new_value": "622208******5678",
        "masked": true
    },{
        "field": "account_name",
        "old_value": "张三",
        "new_value": "张四",
        "masked": false
    }],
    "change_reason": "user_self_update"
}
```

</td>
</tr>
</tbody></table>

- 品牌

    - 创建

    - 审核

<table><tbody>
<tr>
<td>

事件名称

</td>
<td>

品牌变更

**star\.user\.brand\.update**

</td>
<td>

事件类型

</td>
<td>

eventbus

</td>
</tr>
<tr>
<td>

事件描述

</td>
<td colspan="3">

订阅星图账户品牌变更消息

</td>
</tr>
<tr>
<td>

链接

</td>
<td colspan="3">



</td>
</tr>
<tr>
<td>

维护人

</td>
<td>

@何家为

</td>
<td>

qps

</td>
<td>

50

</td>
</tr>
<tr>
<td>

消息结构

与示例



</td>
<td colspan="3">

```Plain Text
{
  "operate_type": 1, # 1 创建 / 2 更新
  "brand_id": 1, # 品牌id
  "account_id": 1, # 关联的账户id
  "audit_status": 1 # 审核状态
  "event_time": 1744636593
}
```

</td>
</tr>
</tbody></table>

- 等级

    - 客户等级升降

    - 机构等级升降

<table><tbody>
<tr>
<td>

事件名称

</td>
<td>

品牌变更

**star\.user\.grade\.update**

</td>
<td>

事件类型

</td>
<td>

eventbus

</td>
</tr>
<tr>
<td>

事件描述

</td>
<td colspan="3">

订阅星图账户等级变更消息

</td>
</tr>
<tr>
<td>

链接

</td>
<td colspan="3">



</td>
</tr>
<tr>
<td>

维护人

</td>
<td>

@何家为

</td>
<td>

qps

</td>
<td>

50

</td>
</tr>
<tr>
<td>

消息结构

与示例



</td>
<td colspan="3">

```Plain Text
{
  "type": 1, # 等级类型, 1: 客户等级，2：机构等级
  "account_id": 1, # 关联的账户id
  "old_level": 1 # 之前等级
  "new_level": 2 # 当前等级
  "event_time": 1744636593
}
```

</td>
</tr>
</tbody></table>

- 身份切换

    - 达人野生\-PGC

<table><tbody>
<tr>
<td>

事件名称

</td>
<td>

身份变更

**star\.user\.financial\.update**

</td>
<td>

事件类型

</td>
<td>

eventbus

</td>
</tr>
<tr>
<td>

事件描述

</td>
<td colspan="3">

订阅星图账户财务身份变更消息

</td>
</tr>
<tr>
<td>

链接

</td>
<td colspan="3">



</td>
</tr>
<tr>
<td>

维护人

</td>
<td>

@何家为

</td>
<td>

qps

</td>
<td>

50

</td>
</tr>
<tr>
<td>

消息结构

与示例



</td>
<td colspan="3">

```Plain Text
enum FinancialRole {
    Normal # 野生
    Individual # 个体户
    Corporate # 对公
}
{
  "account_id": 1, # 关联的账户id
  "old_role": Normal # 之前等级
  "new_role": Individual # 当前等级
  "event_time": 1744636593
}
```

</td>
</tr>
</tbody></table>

- 权限开通

    - 机构承包任务

    - 服务商任务

## 4\.离线数据\(可选\)



## 5\.实时数据流\(可选\)