import React from 'react';
import { LazyLoadComponent } from '@cozeloop/components';

const Demo = () => (
  <div style={{ height: 2000, paddingTop: 1500 }}>
    <LazyLoadComponent>
      <div style={{ padding: 16, background: '#f0f0f0' }}>
        滚动到此处才会加载的内容
      </div>
    </LazyLoadComponent>
  </div>
);

export default Demo;
