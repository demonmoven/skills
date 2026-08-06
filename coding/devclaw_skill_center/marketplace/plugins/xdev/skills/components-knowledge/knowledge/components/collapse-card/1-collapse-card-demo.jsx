import React from 'react';
import { CollapseCard } from '@cozeloop/components';

const Demo = () => (
  <CollapseCard
    title={<span>折叠卡片标题</span>}
    defaultVisible={true}
    subInfo={<span style={{ color: '#999' }}>附加信息</span>}
    extra={<button>操作</button>}
  >
    <div>折叠卡片内容区域</div>
  </CollapseCard>
);

export default Demo;
