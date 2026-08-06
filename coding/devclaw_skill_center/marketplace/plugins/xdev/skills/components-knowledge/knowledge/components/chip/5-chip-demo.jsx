import { Chip } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', gap: '8px' }}>
    <Chip disabled color="brand">
      disabled
    </Chip>
    <Chip disabled chipStyle="remove" color="primary">
      disabled
    </Chip>
    <Chip disabled chipStyle="select" color="green">
      disabled
    </Chip>
  </div>
);

export default Demo;