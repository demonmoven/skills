import { Button } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', gap: 8 }}>
    <Button>默认按钮</Button>
    <Button color="brand">品牌按钮</Button>
    <Button color="highlight">高亮按钮</Button>
    <Button disabled>禁用按钮</Button>
  </div>
);

export default Demo;