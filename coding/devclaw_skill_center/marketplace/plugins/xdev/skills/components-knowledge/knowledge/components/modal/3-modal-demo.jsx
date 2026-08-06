import { Modal, Button } from '@coze-arch/coze-design';
import { useState } from 'react';

const Demo = () => {
  const [visible, setVisible] = useState({});
  const heights = [
    { height: 'fit-content', content: '适应内容高度' },
    { height: 'fill', content: '填充可用空间' },
    { height: 300, content: '固定高度 300px' },
  ];

  return (
    <div className="flex gap-2 flex-wrap">
      {heights.map(({ height, content }) => (
        <>
          <Button onClick={() => setVisible({ ...visible, [height]: true })}>
            高度: {height}
          </Button>
          <Modal
            type="modal"
            height={height}
            title="对话框标题"
            visible={visible[height]}
            onOk={() => setVisible({ ...visible, [height]: false })}
            onCancel={() => setVisible({ ...visible, [height]: false })}
            cancelText="取消"
            okText="确定"
          >
            {content}
          </Modal>
        </>
      ))}
    </div>
  );
};

export default Demo;