import { useState } from 'react';
import { Radio } from '@coze-arch/coze-design';

const Demo = () => {
  const [value, setValue] = useState(1);

  return (
    <Radio.Group
      value={value}
      onChange={e => setValue(e.target.value)}
      direction="horizontal"
    >
      <Radio value={1}>选项A</Radio>
      <Radio value={2}>选项B</Radio>
      <Radio value={3}>选项C</Radio>
    </Radio.Group>
  );
};

export default Demo;