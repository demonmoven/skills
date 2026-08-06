import { Modal, Button } from '@coze-arch/coze-design';
import { useState } from 'react';

const Demo = () => {
  const [visible, setVisible] = useState(false);

  return (
    <>
      <Button onClick={() => setVisible(true)}>显示加载状态</Button>
      <Modal
        type="modal"
        title="对话框标题"
        visible={visible}
        onOk={() => setVisible(false)}
        onCancel={() => setVisible(false)}
        cancelText="取消"
        okText="确定"
        confirmLoading
        cancelLoading
      >
        按钮处于加载状态
      </Modal>
    </>
  );
};

export default Demo;