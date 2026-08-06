import React, { useState } from 'react';
import { MultipartEditor } from '@cozeloop/components';

const Demo = () => {
  const [items, setItems] = useState([]);

  return (
    <MultipartEditor
      spaceID="demo-space"
      value={items}
      onChange={setItems}
    />
  );
};

export default Demo;
