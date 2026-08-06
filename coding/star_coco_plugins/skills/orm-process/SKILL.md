---
name: orm-process
description: 指导orm层的开发，包括model、dao层的代码生成和集成；在db发现变更时，必须依据此skill进行开发
version: 1.0.0
author: yehaowei.beicheng
---

# Overview

orm_process 指导orm层的开发，包括model、dao层的代码生成和集成；在db发现变更时，必须依据此skill进行开发

# Steps

## 1.确定dal目录
- 观察当前仓库目录结构，确定dal、model所属目录，一般情况下，仓库的dal层路径为:
  - infrastructure
    - mysql
  或
    - dal
      - dao
      - db
      - model
      - base_dal


## 2. 生成model，dao文件
- 依据design中的sql设计，模仿当前仓库的model文件格式，生成对应的model文件
- 生成对应的dao层文件
- 如果仓库存在base_dal,则模仿现有文件，生成dal文件

## 3. 编辑database.yml
- 若为新增表，则在conf/database.yml中进行注册

### Reference
- 你[必须]完全参照`references`中的范例，并选择合适的dal格式



