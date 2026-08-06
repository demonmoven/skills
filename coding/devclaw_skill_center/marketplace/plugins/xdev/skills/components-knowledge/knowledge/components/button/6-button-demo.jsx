import { Button } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
    <Button size="large">大号按钮</Button>
    <Button size="default">默认按钮</Button>
    <Button size="small">小号按钮</Button>
    <Button size="mini">迷你按钮</Button>
  </div>
);

export default Demo;