import React from 'react';
import { JumpIconButton } from '@cozeloop/components';

const Demo = () => (
  <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
    <span>查看详情</span>
    <JumpIconButton onClick={() => window.open('https://example.com')} />
  </div>
);

export default Demo;
