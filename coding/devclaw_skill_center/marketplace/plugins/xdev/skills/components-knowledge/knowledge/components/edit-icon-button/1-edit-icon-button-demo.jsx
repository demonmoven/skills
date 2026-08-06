import React from 'react';
import { EditIconButton } from '@cozeloop/components';

const Demo = () => (
  <div style={{ display: 'flex', gap: 16, alignItems: 'center' }}>
    <EditIconButton onClick={() => alert('编辑')} />
    <EditIconButton disabled onClick={() => alert('不会触发')} />
  </div>
);

export default Demo;
