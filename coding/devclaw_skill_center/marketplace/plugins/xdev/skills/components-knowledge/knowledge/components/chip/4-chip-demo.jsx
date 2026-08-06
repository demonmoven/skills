import { Chip } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
    <div>
      <h4>只读样式</h4>
      <div style={{ display: 'flex', gap: '8px' }}>
        <Chip chipStyle="readonly" color="brand">
          readonly
        </Chip>
        <Chip chipStyle="readonly" color="primary">
          readonly
        </Chip>
      </div>
    </div>
    <div>
      <h4>可移除样式</h4>
      <div style={{ display: 'flex', gap: '8px' }}>
        <Chip chipStyle="remove" color="green">
          remove
        </Chip>
        <Chip chipStyle="remove" color="yellow">
          remove
        </Chip>
      </div>
    </div>
    <div>
      <h4>可选择样式</h4>
      <div style={{ display: 'flex', gap: '8px' }}>
        <Chip chipStyle="select" color="blue">
          select
        </Chip>
        <Chip chipStyle="select" color="purple">
          select
        </Chip>
      </div>
    </div>
  </div>
);

export default Demo;