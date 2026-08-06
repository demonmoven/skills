import React, { useState } from 'react';
import { RadioButton } from '@cozeloop/components';

const Demo = () => {
  const [value, setValue] = useState('auto');

  return (
    <RadioButton
      value={value}
      onChange={setValue}
      options={[
        { label: '自动', value: 'auto' },
        { label: '手动', value: 'manual' },
        { label: '关闭', value: 'off', disabled: true },
      ]}
    />
  );
};

export default Demo;
