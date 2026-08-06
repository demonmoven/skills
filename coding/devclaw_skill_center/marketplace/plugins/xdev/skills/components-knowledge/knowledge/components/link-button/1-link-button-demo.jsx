import React from 'react';
import { LinkButton } from '@cozeloop/components';

const Demo = () => (
  <div style={{ display: 'flex', gap: 16 }}>
    <LinkButton tooltip="点击操作" onClick={() => alert('clicked')}>
      普通链接按钮
    </LinkButton>
    <LinkButton
      tooltip="异步操作"
      onAsyncClick={() => new Promise((resolve) => setTimeout(resolve, 1000))}
    >
      异步链接按钮
    </LinkButton>
  </div>
);

export default Demo;
