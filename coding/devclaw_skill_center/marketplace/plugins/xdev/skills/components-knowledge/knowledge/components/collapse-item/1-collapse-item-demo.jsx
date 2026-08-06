import React from 'react';
import { CollapseItem } from '@cozeloop/components';

const Demo = () => (
  <CollapseItem title="展开详情" open={true}>
    <div style={{ padding: 8 }}>这是可折叠的内容区域</div>
  </CollapseItem>
);

export default Demo;
