import React from 'react';
import { IDRender } from '@cozeloop/components';

const Demo = () => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
    <IDRender id="7394857261034928" showSuffixLength={5} enableCopy />
    <IDRender id="7394857261034928" useTag enableCopy />
  </div>
);

export default Demo;
