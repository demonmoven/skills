import React from 'react';
import { TableHeader } from '@cozeloop/components';

const Demo = () => (
  <TableHeader
    filterForm={<input placeholder="搜索..." style={{ width: 200 }} />}
    actions={<button>新建</button>}
    refreshButtonPros={{ onClick: () => alert('刷新') }}
  />
);

export default Demo;
