---
name: thrift-idl-process
description: 指导thrift idl的提交、推送、桩代码生成、go依赖下载，是idl开发的唯一标准
version: 1.0.0
author: yehaowei.beicheng
---

# 概述

thrift-idl-process 规定了thrift idl的提交、推送、桩代码(kitex_gen,overpass)生成、go依赖下载，当需求开发涉及到idl定义时，必须使用此skill


# 核心原则
- [必须]遵守团队的idl规范`references/idl_specification.md`
- [必须]基于kiteX rpc框架，kiteX是字节跳动内部框架
- [必须]使用`star_thrift_gen`模式(bytedance-mcp-overpass)或`overpass`模式(bytedance-mcp-star_thrift_gen_mcp)进行桩代码(kiteX_gen)生成，[禁止]本地生成
- 若本次开发涉及idl改动，[必须]需要先完成thrift-idl-process的所有步骤
- [禁止]下载或修改`ad/star_idl_gen`,[只能]在ad/star_idl进行idl的开发
- 使用`TriggerCompileThrift`时，[必须]使用真实操作人的完整作为operator



# 处理步骤

## 1.编写并提交idl
- 进入`../star_idl`仓库编写idl
  - 若../中不存在star_idl仓库，则尝试搜索`star_idl`仓库，若搜索不到，则使用`git clone`方式获取
- 拉取远端master分支，并新建分支
- 在`/idl`目录下找到当前服务对应的子目录，并直接在`_origin.thrift`中编写idl
- 完成idl的定义和编写之后，**push**到**特定分支**
- 使用工具`TriggerCompileThrift`触发`star_idl_gen`生成,等待10s


## 2. 触发桩代码生成并go get
- [必须]使用工具`TriggerCompileThrift`触发`star_kitex_gen`桩代码生成
- [必须]使用工具`generate_psm_repo`触发kiteX_gen桩代码生成
- 进入当前服务的main.go,观察服务端启动所使用的kiteX_gen桩代码包：
  - 如果是`star_kitex_gen`:
    - 等待60s后，执行`go get code.byted.org/ad/star_kitex_gen@` + `指定的tag`
  - 如果是`overpass`：
    - 使用`overpass`的`get_psm_repo_info`工具获取当前psm的`go mod`路径
    - 等待60s后，执行`go get` + `GoModPath@` + `分支名`
### 注意事项
- 若go get失败，则等待20s再次尝试，**最大重试3次**，若仍不成功则放弃并通知用户，[禁止]使用clone等手段获取桩代码
- 使用工具`TriggerCompileThrift`时，入参email可以使用`git config --get user.email`命令获取



