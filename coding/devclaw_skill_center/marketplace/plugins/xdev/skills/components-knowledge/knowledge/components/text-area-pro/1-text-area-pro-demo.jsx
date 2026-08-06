import React, { useState } from 'react';
import { TextAreaPro } from '@cozeloop/components';

const Demo = () => {
  const [value, setValue] = useState('');

  return (
    <TextAreaPro
      value={value}
      onChange={setValue}
      placeholder="支持全屏编辑的文本域"
      rows={4}
    />
  );
};

export default Demo;
