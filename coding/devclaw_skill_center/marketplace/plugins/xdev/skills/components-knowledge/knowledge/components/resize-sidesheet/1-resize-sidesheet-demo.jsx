import React, { useState } from 'react';
import { ResizeSidesheet } from '@cozeloop/components';

const Demo = () => {
  const [visible, setVisible] = useState(false);

  return (
    <div>
      <button onClick={() => setVisible(true)}>打开侧边栏</button>
      <ResizeSidesheet
        visible={visible}
        onCancel={() => setVisible(false)}
        title="可调整宽度"
        showDivider
        dragOptions={{
          minSize: 400,
          maxSize: 900,
          defaultSize: 600,
        }}
      >
        <div style={{ padding: 16 }}>拖拽左侧边缘调整宽度</div>
      </ResizeSidesheet>
    </div>
  );
};

export default Demo;
