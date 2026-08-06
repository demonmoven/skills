import React, { useState } from 'react';
import { CodeEditor } from '@cozeloop/components';

const Demo = () => {
  const [value, setValue] = useState('{\n  "key": "value"\n}');

  return (
    <div style={{ height: 300 }}>
      <CodeEditor
        language="json"
        value={value}
        onChange={(newValue) => setValue(newValue || '')}
        theme="vs-dark"
        options={{
          minimap: { enabled: false },
          automaticLayout: true,
        }}
      />
    </div>
  );
};

export default Demo;
