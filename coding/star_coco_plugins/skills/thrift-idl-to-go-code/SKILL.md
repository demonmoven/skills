---
name: thrift-idl-to-go-code
description: 在编写go代码时，对于涉及到的接口调用或引用idl中的结构体、枚举，thrift-idl-to-go-code能够完成客户端桩代码的生成并找到所需的准确的go客户端桩代码，给出准确的包路径和代码范例。
version: 1.0.0
author: yehaowei.beicheng
---

# 概述

thrift-idl-to-go-code 规定了在thrift客户端桩代码中搜索所需的**方法、结构体、枚举等**
的准确包路径和go代码的步骤，当go代码开发涉及到idl时，必须使用此skill

# 快速开始


## 1.获取完整idl

**如果用户已经提供具体idl，则跳过此步**

- 使用`overpass`的`get_psm_idl_info`工具确定目标psm的idl仓库
- 浏览idl仓库的目录，搜索目标psm的thrift所在目录和文件
- 在目标文件的中**使用grep进行搜索**
- 根据include完整搜索**model、consts等依赖**，直至获取完整idl

## 2. 获取客户端桩代码

### 2.1 获取overpass客户端仓库

- 使用`overpass`的`get_psm_repo_info`工具获取目标psm的idl仓库，获取主要的依赖go代码:
    - 若无法获取到，则使用工具`generate_psm_repo`触发客户端桩代码生成

#### 工具返回解释

- `RPCMainStructImportPath`为overpass客户端代码中结构体的import路径，req、resp等对象在此路径中
- `RPCMethodImportPath`为overpass客户端代码中具体调用方法的import路径，调用方法在此路径中
- `GoModPath`为overpass客户端代码仓库路径

### 2.2 搜索结构体、枚举等依赖的go代码

**若用户明确需要使用，或者req、resp中包含其他结构体、枚举等依赖**，根据步骤2.1中所获得的idl仓库和路径，在客户端仓库中进行搜索

- 使用`go get` + `GoModPath`获取客户端桩代码仓库
- 使用BASH命令获取用户的环境变量: `go env| grep GOPATH`
- 在`{GOPATH}/src`下找到目标客户端桩代码仓库
- 观察客户端桩代码仓库的目录，根据命名初步确定搜索范围
- 在客户端桩代码仓库中使用`grep`搜索结构体、枚举等的明确定义位置，定位到准确的go代码
- 提取出准确的包名和代码，在后续编码工作中使用

# 注意事项

- 客户端桩代码文件非常大，不可全部读取
- 提前判断是否需要特定branch的idl
- 搜索结果是不可控的，若经历10次以上搜索都无法找到目标idl或桩代码，则默认不存在

