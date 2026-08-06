import { Modal, Button } from '@coze-arch/coze-design';
import { useState } from 'react';

const Demo = () => {
  const [visible, setVisible] = useState({});
  const colors = ['brand', 'yellow', 'red'];

  return (
    <div className="flex gap-2 flex-wrap">
      {colors.map(color => (
        <>
          <Button
            color={color}
            onClick={() => setVisible({ ...visible, [color]: true })}
          >
            {color} 按钮
          </Button>
          <Modal
            type="modal"
            title="对话框标题"
            visible={visible[color]}
            onOk={() => setVisible({ ...visible, [color]: false })}
            onCancel={() => setVisible({ ...visible, [color]: false })}
            cancelText="取消"
            okText="确定"
            okButtonColor={color}
          >
            自定义按钮颜色的对话框
          </Modal>
        </>
      ))}
    </div>
  );
};

export default Demo;