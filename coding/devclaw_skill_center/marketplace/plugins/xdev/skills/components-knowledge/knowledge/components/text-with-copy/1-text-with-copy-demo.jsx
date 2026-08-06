import React from 'react';
import { TextWithCopy } from '@cozeloop/components';

const Demo = () => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
    <TextWithCopy content="7394857261034928" maxWidth={200} />
    <TextWithCopy
      content="https://example.com/very-long-url-that-needs-truncation"
      displayText="示例链接"
      copyTooltipText="复制链接"
      maxWidth={150}
    />
  </div>
);

export default Demo;
