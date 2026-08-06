import { Toast, Button } from '@coze-arch/coze-design';
import { IconCozClockFill } from '@coze-arch/coze-design/icons';
import { useState } from 'react';

const Demo = () => {
  const [toastId, setToastId] = useState<string | null>(null);

  const showToast = () => {
    if (toastId) return;
    const id = Toast.info({
      content: '这是一条可手动关闭的提示',
      duration: 0,
      showClose: true,
      onClose: () => setToastId(null),
    });
    setToastId(id);
  };

  const hideToast = () => {
    if (toastId) {
      Toast.close(toastId);
      setToastId(null);
    }
  };

  return (
    <div className="flex gap-2">
      <Button onClick={showToast}>显示</Button>
      <Button color="red" onClick={hideToast}>
        关闭
      </Button>
    </div>
  );
};

export default Demo;