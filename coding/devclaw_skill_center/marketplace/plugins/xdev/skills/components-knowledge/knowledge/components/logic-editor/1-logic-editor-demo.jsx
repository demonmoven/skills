import React, { useState } from 'react';
import { LogicEditor } from '@cozeloop/components';

const fields = [
  {
    key: 'status',
    label: '状态',
    operators: [
      { label: '等于', value: 'eq' },
      { label: '不等于', value: 'neq' },
    ],
    rightType: 'select',
    rightOptions: [
      { label: '成功', value: 'success' },
      { label: '失败', value: 'fail' },
    ],
  },
  {
    key: 'name',
    label: '名称',
    operators: [
      { label: '包含', value: 'contains' },
      { label: '等于', value: 'eq' },
    ],
    rightType: 'input',
  },
];

const Demo = () => {
  const [filter, setFilter] = useState(undefined);

  return (
    <LogicEditor
      fields={fields}
      value={filter}
      onChange={setFilter}
      onConfirm={(val) => console.log('confirmed', val)}
    />
  );
};

export default Demo;
