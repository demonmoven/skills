import React, { useState } from 'react';
import { LoopRadioGroup } from '@cozeloop/components';

const Demo = () => {
  const [value, setValue] = useState('a');

  return (
    <LoopRadioGroup
      value={value}
      onChange={(e) => setValue(e.target.value)}
      options={[
        { label: '选项 A', value: 'a' },
        { label: '选项 B', value: 'b' },
        { label: '选项 C', value: 'c' },
      ]}
    />
  );
};

export default Demo;
