import React, { useState } from 'react';
import { CodeMirrorCodeEditor } from '@cozeloop/components';

const Demo = () => {
  const [value, setValue] = useState('console.log("Hello World");');

  return (
    <div style={{ height: 300 }}>
      <CodeMirrorCodeEditor
        value={value}
        onChange={setValue}
        language="typescript"
        theme="coze-light"
        minHeight={200}
        borderRadius={8}
      />
    </div>
  );
};

export default Demo;
