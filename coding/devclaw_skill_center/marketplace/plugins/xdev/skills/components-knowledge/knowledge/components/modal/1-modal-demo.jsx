import { Modal, Button } from '@coze-arch/coze-design';
import { useState } from 'react';

const Demo = () => {
  const [visible, setVisible] = useState({ modal: false, dialog: false });

  return (
    <>
      <Button onClick={() => setVisible({ ...visible, modal: true })}>
        打开 Modal
      </Button>
      <Button
        className="ml-2"
        onClick={() => setVisible({ ...visible, dialog: true })}
      >
        打开 Dialog
      </Button>

      <Modal
        type="modal"
        title="删除此机器人？"
        visible={visible.modal}
        onOk={() => setVisible({ ...visible, modal: false })}
        onCancel={() => setVisible({ ...visible, modal: false })}
        cancelText="取消"
        okText="删除"
      >
        此操作无法撤销
      </Modal>

      <Modal
        type="dialog"
        title="删除此机器人？"
        visible={visible.dialog}
        onOk={() => setVisible({ ...visible, dialog: false })}
        onCancel={() => setVisible({ ...visible, dialog: false })}
        cancelText="取消"
        okText="删除"
      >
        此操作无法撤销
      </Modal>
    </>
  );
};

export default Demo;