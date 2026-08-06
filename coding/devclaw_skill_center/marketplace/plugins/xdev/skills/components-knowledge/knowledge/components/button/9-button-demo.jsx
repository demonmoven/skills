import { Button } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', gap: 8 }}>
    <Button showBadge>默认徽标</Button>
    <Button showBadge badgeColor="unset">
      自定义徽标颜色
    </Button>
  </div>
);

export default Demo;