import { useState } from 'react';
import { Popover, Button } from '@coze-arch/coze-design';

const Demo = () => {
  const [visible, setVisible] = useState(false);

  return (
    <Popover
      content="这是一个受控的 Popover"
      trigger="custom"
      visible={visible}
      onVisibleChange={setVisible}
    >
      <Button onClick={() => setVisible(!visible)}>
        {visible ? '点击关闭' : '点击打开'}
      </Button>
    </Popover>
  );
};

export default Demo;