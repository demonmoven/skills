# 组件列表

所有组件的文档和 Demo 统一存放在 `knowledge/components/<组件目录>/` 下，每个目录包含 `config.json`（组件描述和 demo 列表）和 demo 代码文件。

查找组件时，根据下表找到对应目录，读取 config.json 和 demo/文档文件即可。

| 组件 | 引入方式 | 描述 | 文档路径 |
|-----|---------|-----|---------|
| Anchor | `import { Anchor } from '@coze-arch/coze-design'` | 创建超链接导航栏。 | knowledge/components/anchor/ |
| AudioRender | `import { AudioRender } from '@cozeloop/components'` | 音频渲染组件，用于多模态数据集中的音频条目展示。 | knowledge/components/audio-render/ |
| AutoComplete | `import { AutoComplete } from '@coze-arch/coze-design'` | 输入框自动填充。 | knowledge/components/autocomplete/ |
| Avatar | `import { Avatar } from '@coze-arch/coze-design'` | 头像组件：用于展示用户、机器人、平台等的头像图标，支持多种尺寸和类型。 | knowledge/components/avatar/ |
| BackTop | `import { BackTop } from '@coze-arch/coze-design'` | BackTop 回到顶部。 | knowledge/components/backtop/ |
| Badge | `import { Badge } from '@coze-arch/coze-design'` | 徽标组件：用于展示数字、文本或状态标记。 | knowledge/components/badge/ |
| Banner | `import { Banner } from '@coze-arch/coze-design'` | 横幅组件：用于向用户显示常驻提示信息。 | knowledge/components/banner/ |
| BaseSearchSelect | `import { BaseSearchSelect } from '@cozeloop/components'` | 带远程搜索和缓存能力的 Select 选择器。 | knowledge/components/base-search-select/ |
| BasicCard | `import { BasicCard } from '@cozeloop/components'` | 基础卡片组件，包含标题栏和内容区域，带圆角边框样式。 | knowledge/components/basic-card/ |
| Breadcrumb | `import { Breadcrumb } from '@coze-arch/coze-design'` | 面包屑组件：显示当前页面在系统层级结构中的位置。 | knowledge/components/breadcrumb/ |
| Button | `import { Button } from '@coze-arch/coze-design'` | 按钮组件：用于触发即时操作的交互元素。 | knowledge/components/button/ |
| Calendar | `import { Calendar } from '@coze-arch/coze-design'` | 日历组件，允许以日/周/月视图展示对应事件。 | knowledge/components/calendar/ |
| Card | `import { Card } from '@coze-arch/coze-design'` | 常规的卡片容器，可以承载标题、段落、图片、列表等内容。 | knowledge/components/card/ |
| CardPane | `import { CardPane } from '@cozeloop/components'` | 卡片面板容器，支持 hover 时显示阴影效果。 | knowledge/components/card-pane/ |
| Carousel | `import { Carousel } from '@coze-arch/coze-design'` | 轮播图组件，展示多张图片轮流播放的效果。 | knowledge/components/carousel/ |
| Cascader | `import { Cascader } from '@coze-arch/coze-design'` | 级联选择组件：多层级选择，适用于省市区、组织架构等场景。 | knowledge/components/cascader/ |
| Checkbox | `import { Checkbox } from '@coze-arch/coze-design'` | 复选框组件：在一组选项中进行多项选择。 | knowledge/components/checkbox/ |
| Chip | `import { Chip } from '@coze-arch/coze-design'` | 按钮标签组件：展示标签、状态、分类等信息。 | knowledge/components/chip/ |
| ChipSelect | `import { ChipSelect } from '@cozeloop/components'` | 基于 Coze Design Select 的 Chip 样式选择器封装。 | knowledge/components/chip-select/ |
| CodeEditor | `import { CodeEditor } from '@cozeloop/components'` | 对 @monaco-editor/react 的封装，提供 Monaco CodeEditor 和 DiffEditor。 | knowledge/components/code-editor/ |
| CodeEditorWithLoading | `import { CodeEditorWithLoading } from '@cozeloop/components'` | 带 Loading 状态的 Monaco JSON 代码编辑器。 | knowledge/components/code-editor-with-loading/ |
| CodeMirrorCodeEditor | `import { CodeMirrorCodeEditor } from '@cozeloop/components'` | 基于 CodeMirror 的代码编辑器，同时导出 Text/JSON 等变体。 | knowledge/components/codemirror-editor/ |
| Collapse | `import { Collapse } from '@coze-arch/coze-design'` | 折叠面板组件：将复杂内容区域分组和隐藏。 | knowledge/components/collapse/ |
| CollapseCard | `import { CollapseCard } from '@cozeloop/components'` | 可折叠卡片组件，支持受控/非受控模式。 | knowledge/components/collapse-card/ |
| CollapseItem | `import { CollapseItem } from '@cozeloop/components'` | 简易的可折叠条目组件，点击标题切换展开/收起。 | knowledge/components/collapse-item/ |
| Collapsible | `import { Collapsible } from '@coze-arch/coze-design'` | 行为组件，用于展开或折叠内容的容器。 | knowledge/components/collapsible/ |
| ColumnSelector | `import { ColumnSelector } from '@cozeloop/components'` | 表格列选择器，支持列的勾选显示/隐藏和拖拽排序。 | knowledge/components/column-selector/ |
| Copyable | `import { Copyable } from '@cozeloop/components'` | 可复制文本组件，附带复制图标按钮。 | knowledge/components/copyable/ |
| DatePicker | `import { DatePicker } from '@coze-arch/coze-design'` | 日期选择器组件：快速选择日期或日期范围。 | knowledge/components/date-picker/ |
| Descriptions | `import { Descriptions } from '@coze-arch/coze-design'` | 描述列表用于键值对的呈现。 | knowledge/components/descriptions/ |
| Divider | `import { Divider } from '@coze-arch/coze-design'` | 分割线组件，用于有逻辑的组织元素内容和页面结构。 | knowledge/components/divider/ |
| EditIconButton | `import { EditIconButton } from '@cozeloop/components'` | 铅笔图标编辑按钮，支持 disabled 状态。 | knowledge/components/edit-icon-button/ |
| EmptyState | `import { EmptyState } from '@coze-arch/coze-design'` | 空状态组件：展示无数据或操作结果反馈。 | knowledge/components/empty-state/ |
| FooterActions | `import { FooterActions } from '@cozeloop/components'` | 底部操作栏组件，提供"取消"和"确认"按钮布局。 | knowledge/components/footer-actions/ |
| Form | `import { Form, FormInput } from '@coze-arch/coze-design'` | 表单组件：收集、验证和提交用户输入。 | knowledge/components/form/ |
| Highlight | `import { Highlight } from '@coze-arch/coze-design'` | 高亮特定内容。 | knowledge/components/highlight/ |
| IDRender | `import { IDRender } from '@cozeloop/components'` | ID 渲染组件，截取末尾显示，hover 展示完整 ID，支持复制。 | knowledge/components/id-render/ |
| Illustration | `import { Illustration } from '@coze-arch/coze-design'` | 插画组件：提供丰富的视觉元素。 | knowledge/components/illustration/ |
| Image | `import { Image } from '@coze-arch/coze-design'` | 用于展示和预览图片。 | knowledge/components/image/ |
| IndexControllerView | `import { IndexControllerView } from '@cozeloop/components'` | 记录索引导航控制器，提供"上一条/下一条"按钮。 | knowledge/components/index-controller/ |
| InfiniteScrollTable | `import { InfiniteScrollTable } from '@cozeloop/components'` | 无限滚动表格组件，滚动加载更多数据。 | knowledge/components/infinite-scroll-table/ |
| InfoTooltip | `import { InfoTooltip } from '@cozeloop/components'` | 信息提示图标组件，hover 展示说明内容。 | knowledge/components/info-tooltip/ |
| Input | `import { Input } from '@coze-arch/coze-design'` | 输入框组件：收集用户文本输入的基础表单控件。 | knowledge/components/input/ |
| InputCode | `import { InputCode } from '@coze-arch/coze-design'` | 验证码输入框组件：分段输入的交互体验。 | knowledge/components/input-code/ |
| InputNumber | `import { InputNumber } from '@coze-arch/coze-design'` | 数字输入框组件：支持按钮调节、键盘操作。 | knowledge/components/input-number/ |
| InputSlider | `import { InputSlider } from '@cozeloop/components'` | 滑块+数字输入联动组件，Slider 与 InputNumber 双向同步。 | knowledge/components/input-slider/ |
| InputWithCount | `import { InputWithCount } from '@cozeloop/components'` | 带字符计数的输入框组件。 | knowledge/components/input-with-count/ |
| JumpIconButton | `import { JumpIconButton } from '@cozeloop/components'` | 跳转图标按钮，表示外链跳转操作。 | knowledge/components/jump-button/ |
| LargeTxtRender | `import { LargeTxtRender } from '@cozeloop/components'` | 大文本渲染组件，分块按需加载渲染。 | knowledge/components/large-txt-render/ |
| Layout | `import { Layout } from '@cozeloop/components'` | 页面布局组件，组合导出 Header、Content、Tabs 三个子模块。 | knowledge/components/loop-layout/ |
| LazyLoadComponent | `import { LazyLoadComponent } from '@cozeloop/components'` | 懒加载组件，未进入可视区域时显示 Skeleton。 | knowledge/components/lazy-load-component/ |
| LinkButton | `import { LinkButton } from '@cozeloop/components'` | 链接样式按钮，支持同步/异步点击。 | knowledge/components/link-button/ |
| Loading | `import { Loading } from '@coze-arch/coze-design'` | 加载组件：在页面加载过程中给予用户反馈。 | knowledge/components/loading/ |
| LogicEditor | `import { LogicEditor } from '@cozeloop/components'` | 逻辑筛选编辑器组件，构建条件过滤表达式。 | knowledge/components/logic-editor/ |
| LogicExpr | `import { LogicExpr } from '@cozeloop/components'` | 逻辑表达式组件，支持嵌套的表达式组编辑。 | knowledge/components/logic-expr/ |
| LoopRadioGroup | `import { LoopRadioGroup } from '@cozeloop/components'` | Loop 项目风格的 RadioGroup 封装。 | knowledge/components/loop-radio-group/ |
| LoopTable | `import { LoopTable } from '@cozeloop/components'` | Loop 项目通用表格组件，预设空状态样式。 | knowledge/components/loop-table/ |
| LoopTabs | `import { LoopTabs } from '@cozeloop/components'` | Loop 项目风格的 Tabs 组件封装。 | knowledge/components/loop-tabs/ |
| Menu | `import { Menu } from '@coze-arch/coze-design'` | 菜单组件：提供操作或选项列表。 | knowledge/components/menu/ |
| Modal | `import { Modal } from '@coze-arch/coze-design'` | 对话框组件：临时覆盖层，显示信息或获取输入。 | knowledge/components/modal/ |
| MultipartEditor | `import { MultipartEditor } from '@cozeloop/components'` | 多部分内容编辑器，支持文本、图片、视频混排编辑。 | knowledge/components/multipart-editor/ |
| Notification | `import { Notification } from '@coze-arch/coze-design'` | 通知用于主动向用户发出消息通知。 | knowledge/components/notification/ |
| OpenDetailButton | `import { OpenDetailButton } from '@cozeloop/components'` | "查看详情"按钮，新窗口打开指定 URL。 | knowledge/components/open-detail-button/ |
| OverflowList | `import { OverflowList } from '@coze-arch/coze-design'` | 自适应展示尽可能多的列表项。 | knowledge/components/overflowlist/ |
| PageContent | `import { PageError, PageLoading, PageNoAuth, PageNotFound } from '@cozeloop/components'` | 页面状态组件集合（加载、404、错误、无权限）。 | knowledge/components/page-content/ |
| Pagination | `import { Pagination } from '@coze-arch/coze-design'` | 分页组件：多页数据导航。 | knowledge/components/pagination/ |
| Popconfirm | `import { Popconfirm } from '@coze-arch/coze-design'` | 气泡确认框组件：轻量级确认对话框。 | knowledge/components/popconfirm/ |
| Popover | `import { Popover } from '@coze-arch/coze-design'` | 气泡卡片组件：展示临时的上下文信息。 | knowledge/components/popover/ |
| PrimaryPage | `import { PrimaryPage } from '@cozeloop/components'` | 一级页面布局组件，标准页面骨架。 | knowledge/components/primary-page/ |
| PrimaryTitle | `import { PrimaryTitle } from '@cozeloop/components'` | 主标题文本组件。 | knowledge/components/primary-title/ |
| Progress | `import { Progress } from '@coze-arch/coze-design'` | 进度条组件：展示操作的当前进度和状态。 | knowledge/components/progress/ |
| Radio | `import { Radio } from '@coze-arch/coze-design'` | 单选框组件：在一组互斥选项中选择一个。 | knowledge/components/radio/ |
| RadioButton | `import { RadioButton } from '@cozeloop/components'` | 自定义单选按钮组，卡片式按钮排列。 | knowledge/components/radio-button/ |
| Rating | `import { Rating } from '@coze-arch/coze-design'` | 展示评分的组件。 | knowledge/components/rating/ |
| ResizableSideSheet | `import { ResizableSideSheet } from '@cozeloop/components'` | 可拖拽调整宽度的侧边抽屉组件（left 模式）。 | knowledge/components/resizable-side-sheet/ |
| ResizeSidesheet | `import { ResizeSidesheet } from '@cozeloop/components'` | 可拖拽调整宽度的侧边抽屉组件，自定义标题栏。 | knowledge/components/resize-sidesheet/ |
| SchemaEditor | `import { SchemaEditor } from '@cozeloop/components'` | Schema 编辑器组件，JSON/代码/纯文本模式切换。 | knowledge/components/schema-editor/ |
| ScrollList | `import { ScrollList } from '@coze-arch/coze-design'` | 滚动列表。 | knowledge/components/scrolllist/ |
| Search | `import { Search } from '@coze-arch/coze-design'` | 搜索框组件：输入关键词搜索。 | knowledge/components/search/ |
| SegmentTab | `import { SegmentTab } from '@coze-arch/coze-design'` | 分段器组件：在不同视图间切换。 | knowledge/components/segment-tab/ |
| Select | `import { Select } from '@coze-arch/coze-design'` | 选择器组件：从选项中选择一个或多个。 | knowledge/components/select/ |
| SemiSchemaForm | `import { SemiSchemaForm } from '@cozeloop/components'` | 基于 RJSF 的 Schema 驱动表单组件。 | knowledge/components/semi-schema-form/ |
| SentinelForm | `import { SentinelForm } from '@cozeloop/components'` | 带监控哨兵能力的表单组件，增加埋点上报。 | knowledge/components/sentinel-form/ |
| SideSheet | `import { SideSheet } from '@coze-arch/coze-design'` | 可从屏幕边沿滑出的浮层面板。 | knowledge/components/sidesheet/ |
| SingleSelect | `import { SingleSelect } from '@coze-arch/coze-design'` | 单选框组件：自定义布局和图标展示。 | knowledge/components/single-select/ |
| Skeleton | `import { Skeleton } from '@coze-arch/coze-design'` | 加载占位组件。 | knowledge/components/skeleton/ |
| Slider | `import { Slider } from '@coze-arch/coze-design'` | 滑动选择器，拖动交互快速选择数值。 | knowledge/components/slider/ |
| Space | `import { Space } from '@coze-arch/coze-design'` | 设置组件之间的间距。 | knowledge/components/space/ |
| Spin | `import { Spin } from '@coze-arch/coze-design'` | 加载器组件。 | knowledge/components/spin/ |
| Step | `import { Step } from '@coze-arch/coze-design'` | 步骤条组件：展示任务的分步骤流程。 | knowledge/components/step/ |
| StepNav | `import { StepNav } from '@cozeloop/components'` | 步骤导航组件，已完成步骤显示勾选图标。 | knowledge/components/step-nav/ |
| Steps | `import { Steps } from '@coze-arch/coze-design'` | 引导用户按规定流程操作的步骤组件。 | knowledge/components/steps/ |
| Switch | `import { Switch } from '@coze-arch/coze-design'` | 开关组件：表示两种状态之间的切换。 | knowledge/components/switch/ |
| TabBar | `import { TabBar } from '@coze-arch/coze-design'` | 标签栏组件：在不同视图之间切换。 | knowledge/components/tab-bar/ |
| Table | `import { Table } from '@coze-arch/coze-design'` | 表格组件：以行列形式展示结构化数据。 | knowledge/components/table/ |
| TableBatchOperate | `import { TableBatchOperate } from '@cozeloop/components'` | 表格批量操作栏组件。 | knowledge/components/table-batch-operate/ |
| TableColActions | `import { TableColActions } from '@cozeloop/components'` | 表格列操作按钮组，超出部分收纳到下拉菜单。 | knowledge/components/table-col-actions/ |
| TableColsConfig | `import { TableColsConfig } from '@cozeloop/components'` | 表格列配置面板组件，支持本地存储记忆。 | knowledge/components/table-cols-config/ |
| TableEmpty | `import { TableEmpty } from '@cozeloop/components'` | 表格空状态组件。 | knowledge/components/table-empty/ |
| TableHeader | `import { TableHeader } from '@cozeloop/components'` | 表格头部工具栏组件。 | knowledge/components/table-header/ |
| TableWithPagination | `import { TableWithPagination } from '@cozeloop/components'` | 带分页的表格组件。 | knowledge/components/table-with-pagination/ |
| Tabs | `import { Tabs, TabPane } from '@coze-arch/coze-design'` | Tabs 标签栏，在不同组/页之间切换。 | knowledge/components/tabs/ |
| Tag | `import { Tag } from '@coze-arch/coze-design'` | 标签组件：标记和分类内容。 | knowledge/components/tag/ |
| TextArea | `import { TextArea } from '@coze-arch/coze-design'` | 多行文本框组件。 | knowledge/components/textarea/ |
| TextAreaPro | `import { TextAreaPro } from '@cozeloop/components'` | 增强版文本域组件，支持全屏编辑。 | knowledge/components/text-area-pro/ |
| TextWithCopy | `import { TextWithCopy } from '@cozeloop/components'` | 带复制功能的文本显示组件。 | knowledge/components/text-with-copy/ |
| TimePicker | `import { TimePicker } from '@coze-arch/coze-design'` | 时间选择器组件。 | knowledge/components/time-picker/ |
| Timeline | `import { Timeline } from '@coze-arch/coze-design'` | 时间轴组件：对信息进行时间排序展示。 | knowledge/components/timeline/ |
| TitleWithSub | `import { TitleWithSub } from '@cozeloop/components'` | 主副标题组合组件。 | knowledge/components/title-with-sub/ |
| Toast | `import { Toast } from '@coze-arch/coze-design'` | 轻提示组件：非阻塞方式提醒用户。 | knowledge/components/toast/ |
| Tooltip | `import { Tooltip } from '@coze-arch/coze-design'` | 文字提示组件：交互时展示简短提示。 | knowledge/components/tooltip/ |
| TooltipWhenDisabled | `import { TooltipWhenDisabled } from '@cozeloop/components'` | 条件 Tooltip 组件，仅在 disabled 时显示提示。 | knowledge/components/tooltip-when-disabled/ |
| TooltipWithDisabled | `import { TooltipWithDisabled } from '@cozeloop/components'` | 可禁用的 Tooltip 组件。 | knowledge/components/tooltip-with-disabled/ |
| Transfer | `import { Transfer } from '@coze-arch/coze-design'` | 多选选择器，支持搜索功能。 | knowledge/components/transfer/ |
| Tree | `import { Tree } from '@coze-arch/coze-design'` | 树型结构列表。 | knowledge/components/tree/ |
| TreeSelect | `import { TreeSelect } from '@coze-arch/coze-design'` | 树形选择器组件：层级关系数据选择。 | knowledge/components/tree-select/ |
| Typography | `import { Typography } from '@coze-arch/coze-design'` | 排版组件：展示不同类型的文本内容。 | knowledge/components/typography/ |
| Upload | `import { Upload } from '@coze-arch/coze-design'` | 文件选择上传。 | knowledge/components/upload/ |
| UserProfile | `import { UserProfile } from '@cozeloop/components'` | 用户信息展示组件，显示头像和用户名。 | knowledge/components/user-profile/ |
| VersionList | `import { VersionList } from '@cozeloop/components'` | 版本列表组件，支持选中高亮和加载更多。 | knowledge/components/version-list/ |
| VideoRender | `import { VideoRender } from '@cozeloop/components'` | 视频渲染组件，用于多模态数据集中的视频条目展示。 | knowledge/components/video-render/ |
