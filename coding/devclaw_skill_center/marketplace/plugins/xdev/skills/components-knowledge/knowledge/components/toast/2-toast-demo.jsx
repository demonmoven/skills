import { Toast, Button } from '@coze-arch/coze-design';
import { IconCozClockFill } from '@coze-arch/coze-design/icons';

const Demo = () => (
  <div className="flex gap-2">
    <Button onClick={() => Toast.info('这是一条信息')}>信息</Button>
    <Button color="green" onClick={() => Toast.success('操作成功')}>
      成功
    </Button>
    <Button color="yellow" onClick={() => Toast.warning('注意警告')}>
      警告
    </Button>
    <Button color="red" onClick={() => Toast.error('发生错误')}>
      错误
    </Button>
  </div>
);

export default Demo;