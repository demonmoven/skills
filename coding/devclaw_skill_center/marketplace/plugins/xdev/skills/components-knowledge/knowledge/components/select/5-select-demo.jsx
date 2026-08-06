import { Select } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
    <div>
      <h4>禁用状态</h4>
      <Select
        placeholder="请选择"
        style={{ width: '200px' }}
        disabled
        optionList={[
          { value: '1', label: '选项1' },
          { value: '2', label: '选项2' },
          { value: '3', label: '选项3' },
        ]}
      />
    </div>
    <div>
      <h4>错误状态</h4>
      <Select
        placeholder="请选择"
        style={{ width: '200px' }}
        hasError
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