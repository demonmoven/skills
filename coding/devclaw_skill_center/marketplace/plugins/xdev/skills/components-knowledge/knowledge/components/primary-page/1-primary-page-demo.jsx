import React from 'react';
import { PrimaryPage } from '@cozeloop/components';

const Demo = () => (
  <PrimaryPage
    pageTitle="页面标题"
    titleSlot={<button>操作按钮</button>}
    filterSlot={<div>筛选区域</div>}
  >
    <div style={{ height: 300, background: '#f5f5f5', borderRadius: 8, padding: 16 }}>
      页面主体内容
    </div>
  </PrimaryPage>
);

export default Demo;
