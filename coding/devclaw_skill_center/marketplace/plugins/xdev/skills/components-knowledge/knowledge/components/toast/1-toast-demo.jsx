import { Toast, Button } from '@coze-arch/coze-design';
import { IconCozClockFill } from '@coze-arch/coze-design/icons';

const Demo = () => (
  <div className="flex gap-2">
    <Button
      onClick={() =>
        Toast.info({
          content: '这是一条提示信息',
          duration: 3,
        })
      }
    >
      显示提示
    </Button>
  </div>
);

export default Demo;