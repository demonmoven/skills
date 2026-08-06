import React, { useState } from 'react';
import { BaseSearchSelect } from '@cozeloop/components';

const Demo = () => {
  const [value, setValue] = useState(undefined);

  return (
    <BaseSearchSelect
      style={{ width: 300 }}
      value={value}
      onChange={setValue}
      placeholder="搜索选择"
      optionList={[
        { label: '选项一', value: '1' },
        { label: '选项二', value: '2' },
        { label: '选项三', value: '3' },
      ]}
    />
  );
};

export default Demo;
