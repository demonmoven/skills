import React from 'react';
import { LargeTxtRender } from '@cozeloop/components';

const longText = Array.from({ length: 1000 }, (_, i) => `Line ${i + 1}: This is a sample text for large text rendering.\n`).join('');

const Demo = () => (
  <div style={{ height: 300, overflow: 'hidden' }}>
    <LargeTxtRender text={longText} />
  </div>
);

export default Demo;
