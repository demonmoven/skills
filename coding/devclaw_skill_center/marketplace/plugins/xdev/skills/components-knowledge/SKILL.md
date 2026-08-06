---
name: components-knowledge
description: 获取组件库使用知识。当需要使用 @coze-arch/coze-design 或 @cozeloop/components 组件进行 UI 开发时使用。
---

# 组件知识库

提供 120+ 个组件的使用文档、API 参考和 Demo 代码。

## 知识结构

所有组件的文档和 Demo 统一存放在 `knowledge/components/` 目录下

```
knowledge/
├── components/           # 所有 UI 组件（统一目录）
│   └── <component>/
│       ├── config.json   # 组件描述 + 引入方式 + demo 列表
│       ├── *.mdx / *.md  # API 文档（部分组件）
│       └── *-demo.jsx    # Demo 代码
├── icons/                # 图标相关
│   ├── icon/             # Icon 组件文档与 Demo
│   └── icons-index.md    # 完整图标列表（521 个图标组件）
└── components-index.md   # 所有组件的统一索引
```

## 使用方式

### 1. 查看组件索引

读取 [knowledge/components-index.md](knowledge/components-index.md) 获取全部组件列表，包含引入方式、描述和文档路径。

### 2. 获取组件详情

根据索引找到组件路径后：

1. **读取 config.json** — 获取组件名称、描述、引入方式和 demo 列表
2. **读取 demo.jsx** — 获取使用示例代码
3. **读取 mdx/md 文件**（如有） — 获取完整 API 文档


### 3. 图标使用

```tsx
import { IconCozEdit, IconCozPeopleFill } from '@coze-arch/coze-design/icons';

<IconCozEdit />
<IconCozPeopleFill size="large" />
```

完整的可用图标列表见 [knowledge/icons/icons-index.md](knowledge/icons/icons-index.md)（共 521 个图标组件）。
图标组件的用法示例和 API 说明见 [knowledge/icons/icon/](knowledge/icons/icon/)。

## config.json 格式说明

每个组件目录下的 `config.json` 包含：

```json
{
  "name": "Button",
  "description": "按钮组件描述",
  "demo": [
    { "name": "1-button-demo", "description": "基本用法" },
    { "name": "2-button-demo", "description": "图标按钮" }
  ]
}
```

对应文件：
- `1-button-demo.jsx` — Demo 代码
- `1-button-demo.png` — Demo 截图（部分组件提供）
