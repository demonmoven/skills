import { SingleSelect } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
    <div>
      <h4>禁用单个选项</h4>
      <SingleSelect
        defaultValue="2"
        options={[
          { value: '1', label: 'Option 1', disabled: true },
          { value: '2', label: 'Option 2', disabled: true },
          { value: '3', label: 'Option 3' },
        ]}
      />
    </div>
    <div>
      <h4>禁用整个组件</h4>
      <SingleSelect
        disabled
        defaultValue="1"
        options={[
          { value: '1', label: 'Option 1' },
          { value: '2', label: 'Option 2' },
          { value: '3', label: 'Option 3' },
        ]}
      />
    </div>
  </div>
);

export default Demo;