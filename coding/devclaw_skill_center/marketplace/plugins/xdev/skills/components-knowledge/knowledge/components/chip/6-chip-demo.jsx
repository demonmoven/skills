import { Chip } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', gap: '8px' }}>
    <Chip loading color="brand">
      loading
    </Chip>
    <Chip loading chipStyle="remove" color="primary">
      loading
    </Chip>
    <Chip loading chipStyle="select" color="green">
      loading
    </Chip>
  </div>
);

export default Demo;