import { Toast, Button } from '@coze-arch/coze-design';
import { IconCozClockFill } from '@coze-arch/coze-design/icons';

const Demo = () => (
  <div className="flex gap-2">
    <Button
      onClick={() =>
        Toast.info({
          content: '自定义图标和关闭按钮',
          duration: 0,
          showClose: true,
          icon: <IconCozClockFill className="text-brand-6" />,
        })
      }
    >
      自定义配置
    </Button>
  </div>
);

export default Demo;