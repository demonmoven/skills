import React, { useState } from 'react';
import { ChipSelect } from '@cozeloop/components';

const Demo = () => {
  const [value, setValue] = useState('1');

  return (
    <ChipSelect
      style={{ width: 200 }}
      value={value}
      onChange={setValue}
      optionList={[
        { label: 'Option A', value: '1' },
        { label: 'Option B', value: '2' },
        { label: 'Option C', value: '3' },
      ]}
    />
  );
};

export default Demo;
