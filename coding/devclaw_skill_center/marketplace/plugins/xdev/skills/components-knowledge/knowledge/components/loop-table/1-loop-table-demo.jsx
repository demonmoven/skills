import React from 'react';
import { LoopTable } from '@cozeloop/components';

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id' },
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '状态', dataIndex: 'status', key: 'status' },
];

const dataSource = [
  { key: '1', id: '001', name: '任务一', status: '进行中' },
  { key: '2', id: '002', name: '任务二', status: '已完成' },
  { key: '3', id: '003', name: '任务三', status: '待开始' },
];

const Demo = () => (
  <LoopTable
    tableProps={{
      columns,
      dataSource,
    }}
  />
);

export default Demo;
