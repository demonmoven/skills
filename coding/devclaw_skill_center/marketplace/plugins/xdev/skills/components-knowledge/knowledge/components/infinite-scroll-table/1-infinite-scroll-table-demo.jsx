import React from 'react';
import { InfiniteScrollTable } from '@cozeloop/components';

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id' },
  { title: '名称', dataIndex: 'name', key: 'name' },
];

let pageIndex = 0;

const fetchData = async () => {
  pageIndex++;
  const list = Array.from({ length: 20 }, (_, i) => ({
    id: `${(pageIndex - 1) * 20 + i + 1}`,
    name: `Item ${(pageIndex - 1) * 20 + i + 1}`,
    key: `${(pageIndex - 1) * 20 + i + 1}`,
  }));
  return { list, hasMore: pageIndex < 5 };
};

const Demo = () => (
  <div style={{ height: 400 }}>
    <InfiniteScrollTable
      service={fetchData}
      tableProps={{ columns }}
    />
  </div>
);

export default Demo;
