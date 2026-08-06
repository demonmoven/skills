import { Select } from '@coze-arch/coze-design';

const Demo = () => (
  <Select
    placeholder="请选择"
    style={{ width: '200px' }}
    showClear
    optionList={[
      { value: '1', label: '选项1' },
      { value: '2', label: '选项2' },
      { value: '3', label: '选项3' },
    ]}
  />
);

export default Demo;