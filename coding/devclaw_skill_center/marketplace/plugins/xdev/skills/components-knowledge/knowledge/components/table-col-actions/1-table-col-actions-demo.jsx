import React from 'react';
import { TableColActions } from '@cozeloop/components';

const Demo = () => (
  <TableColActions
    maxCount={2}
    actions={[
      { label: '编辑', onClick: () => alert('编辑') },
      { label: '复制', onClick: () => alert('复制') },
      { label: '删除', type: 'danger', onClick: () => alert('删除') },
      { label: '归档', disabled: true, disabledTooltip: '暂不支持归档' },
    ]}
  />
);

export default Demo;
