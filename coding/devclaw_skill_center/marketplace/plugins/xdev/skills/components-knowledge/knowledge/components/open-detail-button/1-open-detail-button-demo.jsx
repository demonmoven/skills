import React from 'react';
import { OpenDetailButton } from '@cozeloop/components';

const Demo = () => (
  <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
    <span>任务名称</span>
    <OpenDetailButton url="https://example.com/detail" />
  </div>
);

export default Demo;
