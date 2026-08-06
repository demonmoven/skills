import React from 'react';
import { TooltipWhenDisabled } from '@cozeloop/components';

const Demo = () => (
  <div style={{ display: 'flex', gap: 16 }}>
    <TooltipWhenDisabled
      disabled={true}
      content="此按钮暂不可用"
      needWrap
    >
      <button disabled>禁用按钮（有提示）</button>
    </TooltipWhenDisabled>
    <TooltipWhenDisabled
      disabled={false}
      content="不会显示"
    >
      <button>启用按钮（无提示）</button>
    </TooltipWhenDisabled>
  </div>
);

export default Demo;
