import { Modal, Button } from '@coze-arch/coze-design';
import { useState } from 'react';

const Demo = () => {
  const [visible, setVisible] = useState(false);

  return (
    <>
      <Button onClick={() => setVisible(true)}>显示自定义内容</Button>
      <Modal
        type="modal"
        title="对话框标题"
        visible={visible}
        onOk={() => setVisible(false)}
        onCancel={() => setVisible(false)}
        cancelText="取消"
        okText="确定"
      >
        <Modal.SubTitle>副标题</Modal.SubTitle>
        <Modal.Description>这是一段描述文本</Modal.Description>
        <Modal.Content>这是主要内容区域</Modal.Content>
      </Modal>
    </>
  );
};

export default Demo;