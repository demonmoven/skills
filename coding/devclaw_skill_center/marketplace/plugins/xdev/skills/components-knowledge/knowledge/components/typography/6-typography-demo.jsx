import { Typography } from '@coze-arch/coze-design';

const { Paragraph } = Typography;

const Demo = () => (
  <div className="flex flex-col gap-4">
    <Paragraph
      fontSize="14px"
      ellipsis={{
        rows: 2,
        showTooltip: {
          type: 'tooltip',
          opts: {
            content: '完整文本内容',
          },
        },
      }}
      style={{ width: 500 }}
    >
      Coze Design
      设计系统包含设计语言以及一整套可复用的前端组件，帮助设计师与开发者更容易地打造高质量的、用户体验一致的、符合设计规范的
      Web 应用。
    </Paragraph>

    <Paragraph
      fontSize="14px"
      ellipsis={{
        rows: 2,
        showTooltip: false,
      }}
      style={{ width: 500 }}
    >
      仅使用省略号：Coze Design
      设计系统包含设计语言以及一整套可复用的前端组件，帮助设计师与开发者更容易地打造高质量的、用户体验一致的、符合设计规范的
      Web 应用。
    </Paragraph>
  </div>
);

export default Demo;