import { Collapse } from '@coze-arch/coze-design';
import { useState } from 'react';

const Demo = () => {
  const [activeKey, setActiveKey] = useState(['1']);

  return (
    <Collapse activeKey={activeKey} onChange={keys => setActiveKey(keys)}>
      <Collapse.Panel header="受控面板 1" itemKey="1">
        面板 1 的内容
      </Collapse.Panel>
      <Collapse.Panel header="受控面板 2" itemKey="2">
        面板 2 的内容
      </Collapse.Panel>
    </Collapse>
  );
};

export default Demo;