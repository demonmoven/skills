import { Popover, Button } from '@coze-arch/coze-design';

const Demo = () => (
  <Popover
    content={
      <article>
        自定义样式的气泡卡片
        <br /> 使用了绿色主题
      </article>
    }
    trigger="click"
    position="bottom"
    showArrow
    style={{
      backgroundColor: 'var(--coz-fg-hglt-green)',
      borderColor: 'var(--coz-fg-hglt-green)',
      color: 'var(--coz-fg-hglt-plus)',
      borderWidth: 1,
      borderStyle: 'solid',
    }}
  >
    <Button color="primary">自定义样式</Button>
  </Popover>
);

export default Demo;