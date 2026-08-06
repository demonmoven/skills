import { Popconfirm, Button } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="space-x-4">
    <Popconfirm
      title="确定要保存更改？"
      content="此操作需要同步到服务器"
      onConfirm={() =>
        new Promise(resolve => {
          setTimeout(() => {
            resolve(true);
          }, 2000);
        })
      }
    >
      <Button>异步保存 (2s)</Button>
    </Popconfirm>

    <Popconfirm
      title="确定要取消操作？"
      content="正在执行的任务将被终止"
      cancelText="取消"
      onCancel={() =>
        new Promise(resolve => {
          setTimeout(() => {
            resolve(true);
          }, 1000);
        })
      }
    >
      <Button>异步取消 (1s)</Button>
    </Popconfirm>
  </div>
);

export default Demo;