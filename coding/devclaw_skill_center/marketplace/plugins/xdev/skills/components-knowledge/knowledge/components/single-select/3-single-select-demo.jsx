import { SingleSelect } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
    <div>
      <h4>默认尺寸</h4>
      <SingleSelect
        size="default"
        layout="fill"
        defaultValue="1"
        options={[
          { value: '1', label: 'Default Size' },
          { value: '2', label: 'Option 2' },
          { value: '3', label: 'Option 3' },
        ]}
      />
    </div>
    <div>
      <h4>小尺寸</h4>
      <SingleSelect
        size="small"
        layout="fill"
        defaultValue="1"
        options={[
          { value: '1', label: 'Small Size' },
          { value: '2', label: 'Option 2' },
          { value: '3', label: 'Option 3' },
        ]}
      />
    </div>
  </div>
);

export default Demo;