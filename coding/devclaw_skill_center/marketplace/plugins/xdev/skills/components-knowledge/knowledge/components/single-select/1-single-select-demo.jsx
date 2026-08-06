import { SingleSelect } from '@coze-arch/coze-design';

const Demo = () => (
  <SingleSelect
    defaultValue="1"
    options={[
      { value: '1', label: 'Option 1' },
      { value: '2', label: 'Option 2' },
      { value: '3', label: 'Option 3' },
    ]}
  />
);

export default Demo;