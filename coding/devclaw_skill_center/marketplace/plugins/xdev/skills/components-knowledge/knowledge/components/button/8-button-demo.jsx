import { Button } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', gap: 8 }}>
    <Button loading>加载中</Button>
    <Button loading useSpinIcon={false}>
      自定义加载图标
    </Button>
  </div>
);

export default Demo;