import { Toast, Button } from '@coze-arch/coze-design';
import { IconCozClockFill } from '@coze-arch/coze-design/icons';

const Demo = () => (
  <div className="flex gap-2">
    <Button
      onClick={() => {
        const id = 'update-toast';
        Toast.info({ content: '初始内容', id });
        setTimeout(() => {
          Toast.success({ content: '更新后的内容', id });
        }, 2000);
      }}
    >
      更新内容
    </Button>
  </div>
);

export default Demo;