import { Popover, Button } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', gap: 8 }}>
    <Popover
      content={
        <article>
          Hi ByteDancer, this is a popover.
          <br /> We have 2 lines.
        </article>
      }
      trigger="click"
      position="bottom"
      showArrow
    >
      <Button>点击触发</Button>
    </Popover>

    <Popover
      content={<article>鼠标悬停显示的内容</article>}
      trigger="hover"
      position="top"
      showArrow
    >
      <Button>悬停触发</Button>
    </Popover>
  </div>
);

export default Demo;