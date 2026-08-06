import React from 'react';
import { InfoTooltip } from '@cozeloop/components';

const Demo = () => (
  <div style={{ display: 'flex', gap: 16, alignItems: 'center' }}>
    <span>标签名</span>
    <InfoTooltip content="这是一段提示信息" />
    <span>问号图标</span>
    <InfoTooltip content="使用问号图标的提示" useQuestion />
  </div>
);

export default Demo;
