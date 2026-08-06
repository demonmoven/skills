import { Modal, Button } from '@coze-arch/coze-design';
import { IconCozLoading } from '@coze-arch/coze-design/icons';

const Demo = () => {
  const showModal = type => {
    Modal[type]({
      title: '操作提示',
      content: '这是一个通过静态方法创建的对话框',
      type: 'dialog',
      cancelText: '取消',
      okText: '确定',
    });
  };

  return (
    <div className="flex gap-2 flex-wrap">
      <Button color="brand" onClick={() => showModal('success')}>
        success
      </Button>
      <Button color="brand" onClick={() => showModal('info')}>
        info
      </Button>
      <Button color="red" onClick={() => showModal('error')}>
        error
      </Button>
      <Button color="yellow" onClick={() => showModal('warning')}>
        warning
      </Button>
      <Button color="brand" onClick={() => showModal('confirm')}>
        confirm
      </Button>
    </div>
  );
};

export default Demo;