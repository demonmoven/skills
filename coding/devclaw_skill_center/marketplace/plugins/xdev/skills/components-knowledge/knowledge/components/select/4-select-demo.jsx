import { Select } from '@coze-arch/coze-design';

const Demo = () => (
  <Select
    placeholder="请选择"
    style={{ width: '300px' }}
    multiple
    showRestTagsPopover
    optionList={[
      { value: '1', label: '选项1' },
      { value: '2', label: '选项2' },
      { value: '3', label: '选项3' },
      { value: '4', label: '选项4' },
      { value: '5', label: '选项5' },
    ]}
  />
);

export default Demo;