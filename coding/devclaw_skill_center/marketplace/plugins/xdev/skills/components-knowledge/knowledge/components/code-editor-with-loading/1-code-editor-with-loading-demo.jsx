import React, { useState } from 'react';
import { CodeEditorWithLoading } from '@cozeloop/components';

const Demo = () => {
  const [value, setValue] = useState('{\n  "name": "example"\n}');

  return (
    <div style={{ height: 300 }}>
      <CodeEditorWithLoading
        value={value}
        onChange={setValue}
        disabled={false}
      />
    </div>
  );
};

export default Demo;
