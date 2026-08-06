import React, { useState } from 'react';
import { ResizableSideSheet } from '@cozeloop/components';

const Demo = () => {
  const [visible, setVisible] = useState(false);

  return (
    <div>
      <button onClick={() => setVisible(true)}>打开抽屉</button>
      <ResizableSideSheet
        visible={visible}
        onCancel={() => setVisible(false)}
        title="可调整宽度的抽屉"
        minSize={300}
        maxSize={800}
        defaultSize={500}
      >
        <div style={{ padding: 16 }}>拖拽左侧边缘可调整宽度</div>
      </ResizableSideSheet>
    </div>
  );
};

export default Demo;
