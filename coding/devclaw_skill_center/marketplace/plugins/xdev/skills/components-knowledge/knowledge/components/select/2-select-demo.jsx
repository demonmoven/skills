import { Select } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', gap: '16px' }}>
    <div>
      <h4>默认尺寸</h4>
      <Select
        placeholder="请选择"
        style={{ width: '200px' }}
        size="default"
        optionList={[
          { value: '1', label: '选项1' },
          { value: '2', label: '选项2' },
          { value: '3', label: '选项3' },
        ]}
      />
    </div>
    <div>
      <h4>小尺寸</h4>
      <Select
        placeholder="请选择"
        style={{ width: '200px' }}
        size="small"
        optionList={[
          { value: '1', label: '选项1' },
          { value: '2', label: '选项2' },
          { value: '3', label: '选项3' },
        ]}
      />
    </div>
  </div>
);

export default Demo;