import React, { useState } from 'react';
import { TableWithPagination } from '@cozeloop/components';

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id' },
  { title: '名称', dataIndex: 'name', key: 'name' },
];

const allData = Array.from({ length: 100 }, (_, i) => ({
  key: String(i + 1),
  id: String(i + 1),
  name: `Item ${i + 1}`,
}));

const Demo = () => {
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const service = {
    data: {
      total: allData.length,
      list: allData.slice((current - 1) * pageSize, current * pageSize),
    },
    pagination: {
      current,
      pageSize,
      total: allData.length,
      onChange: (page, size) => {
        setCurrent(page);
        setPageSize(size);
      },
      changeCurrent: setCurrent,
    },
    loading: false,
  };

  return (
    <TableWithPagination
      service={service}
      tableProps={{ columns }}
    />
  );
};

export default Demo;
