import React, { useState } from 'react';
import { ColumnSelector } from '@cozeloop/components';

const Demo = () => {
  const [columns, setColumns] = useState([
    { key: 'name', value: '名称', checked: true },
    { key: 'status', value: '状态', checked: true },
    { key: 'created_at', value: '创建时间', checked: true },
    { key: 'updated_at', value: '更新时间', checked: false },
  ]);

  return (
    <ColumnSelector
      columns={columns}
      onChange={setColumns}
      sortable={true}
    />
  );
};

export default Demo;
