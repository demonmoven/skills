import { Typography } from '@coze-arch/coze-design';

const { Paragraph } = Typography;

const Demo = () => (
  <Paragraph
    ellipsis={{
      rows: 3,
      expandable: true,
      collapsible: true,
      collapseText: '折叠我吧',
    }}
    className="coz-fg-primary"
    style={{ width: 300 }}
  >
    支持展开和折叠：Coze Design
    设计系统包含设计语言以及一整套可复用的前端组件，帮助设计师与开发者更容易地打造高质量的、用户体验一致的、符合设计规范的
    Web 应用。
  </Paragraph>
);

export default Demo;