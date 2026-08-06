import React from 'react';
import { TooltipWithDisabled } from '@cozeloop/components';

const Demo = () => (
  <div style={{ display: 'flex', gap: 16 }}>
    <TooltipWithDisabled content="Tooltip 内容" disabled={false}>
      <span>悬浮显示提示</span>
    </TooltipWithDisabled>
    <TooltipWithDisabled content="不会显示" disabled={true}>
      <span>Tooltip 已禁用</span>
    </TooltipWithDisabled>
  </div>
);

export default Demo;
