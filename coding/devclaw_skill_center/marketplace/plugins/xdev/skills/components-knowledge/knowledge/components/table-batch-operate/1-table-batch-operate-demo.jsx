import React, { useState } from 'react';
import { TableBatchOperate } from '@cozeloop/components';

const Demo = () => {
  const [selectedItems, setSelectedItems] = useState([]);
  const [enableBatchOperate, setEnableBatchOperate] = useState(false);

  const batchOperateStore = {
    selectedItems,
    setSelectedItems,
    enableBatchOperate,
    setEnableBatchOperate,
  };

  return (
    <TableBatchOperate
      batchOperateStore={batchOperateStore}
      actions={
        <button onClick={() => alert(`删除 ${selectedItems.length} 项`)}>
          批量删除
        </button>
      }
    />
  );
};

export default Demo;
