import React, { useState } from 'react';
import { InputSlider } from '@cozeloop/components';

const Demo = () => {
  const [value, setValue] = useState(0.5);

  return (
    <InputSlider
      value={value}
      onChange={setValue}
      min={0}
      max={1}
      step={0.01}
      decimalPlaces={2}
    />
  );
};

export default Demo;
