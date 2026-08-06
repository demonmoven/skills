import { SegmentTab } from '@coze-arch/coze-design';
import { useState } from 'react';

const Demo = () => {
  const [value, setValue] = useState('Tab1');

  return (
    <SegmentTab
      value={value}
      options={['Tab1', 'Tab2', 'Tab3']}
      onChange={e => {
        console.log('选中值:', e);
        setValue(e.target.value);
      }}
    />
  );
};

export default Demo;