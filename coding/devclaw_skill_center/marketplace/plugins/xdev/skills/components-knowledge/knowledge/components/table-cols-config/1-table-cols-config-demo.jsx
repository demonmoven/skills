import React from 'react';
import { TableColsConfig } from '@cozeloop/components';

const columns = [
  { title: 'ID', dataIndex: 'id', colKey: 'id', configurable: true },
  { title: '名称', dataIndex: 'name', colKey: 'name', configurable: true },
  { title: '状态', dataIndex: 'status', colKey: 'status', configurable: true },
  { title: '创建时间', dataIndex: 'created_at', colKey: 'created_at', configurable: true },
];

const Demo = () => (
  <TableColsConfig
    columns={columns}
    defaultHiddenColKeys={['created_at']}
    localStorageKey="demo-table-cols"
    onChangeConfig={(newCols) => console.log('columns changed', newCols)}
  />
);

export default Demo;
