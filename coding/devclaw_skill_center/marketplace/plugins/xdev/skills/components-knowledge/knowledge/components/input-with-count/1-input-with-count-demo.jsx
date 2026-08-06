import React, { useState } from 'react';
import { InputWithCount } from '@cozeloop/components';

const Demo = () => {
  const [value, setValue] = useState('');

  return (
    <InputWithCount
      value={value}
      onChange={setValue}
      maxLength={50}
      placeholder="最多输入50个字符"
      style={{ width: 300 }}
    />
  );
};

export default Demo;
