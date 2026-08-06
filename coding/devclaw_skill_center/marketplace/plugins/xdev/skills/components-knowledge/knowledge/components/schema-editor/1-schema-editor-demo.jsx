import React, { useState } from 'react';
import { SchemaEditor } from '@cozeloop/components';

const Demo = () => {
  const [value, setValue] = useState('{\n  "type": "object",\n  "properties": {}\n}');

  return (
    <SchemaEditor
      value={value}
      onChange={setValue}
      language="json"
      placeholder="请输入 JSON Schema"
    />
  );
};

export default Demo;
