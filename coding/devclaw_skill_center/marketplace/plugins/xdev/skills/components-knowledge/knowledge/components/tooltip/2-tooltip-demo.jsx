import { Tooltip, Button } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex gap-4">
    <Tooltip content="这是亮色主题" theme="light">
      <Button>亮色主题</Button>
    </Tooltip>
    <Tooltip content="这是暗色主题" theme="dark">
      <Button>暗色主题</Button>
    </Tooltip>
  </div>
);

export default Demo;