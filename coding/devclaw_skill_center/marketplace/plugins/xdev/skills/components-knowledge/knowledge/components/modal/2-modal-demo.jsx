import { Modal, Button } from '@coze-arch/coze-design';
import { useState } from 'react';

const Demo = () => {
  const [visible, setVisible] = useState({});
  const sizes = ['default', 'large', 'xl', 'xxl', 'fill'];

  return (
    <div className="flex gap-2 flex-wrap">
      {sizes.map(size => (
        <>
          <Button onClick={() => setVisible({ ...visible, [size]: true })}>
            {size} 尺寸
          </Button>
          <Modal
            type="modal"
            size={size}
            title="对话框标题"
            visible={visible[size]}
            onOk={() => setVisible({ ...visible, [size]: false })}
            onCancel={() => setVisible({ ...visible, [size]: false })}
            cancelText="取消"
            okText="确定"
          >
            不同尺寸的对话框内容
          </Modal>
        </>
      ))}
    </div>
  );
};

export default Demo;