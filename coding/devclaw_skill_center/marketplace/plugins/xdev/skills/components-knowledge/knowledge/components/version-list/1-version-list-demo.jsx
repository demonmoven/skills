import React, { useState } from 'react';
import { default as VersionList } from '@cozeloop/components';

const versions = [
  { id: '1', version_number: 'v1.0.0', description: '初始版本', created_at: 1700000000 },
  { id: '2', version_number: 'v1.1.0', description: '新增功能', created_at: 1700100000 },
  { id: '3', version_number: 'v2.0.0', description: '重大更新', created_at: 1700200000 },
];

const Demo = () => {
  const [activeId, setActiveId] = useState('3');

  return (
    <VersionList
      versions={versions}
      activeVersionId={activeId}
      onActiveChange={(id) => setActiveId(id)}
      enableLoadMore={false}
    />
  );
};

export default Demo;
