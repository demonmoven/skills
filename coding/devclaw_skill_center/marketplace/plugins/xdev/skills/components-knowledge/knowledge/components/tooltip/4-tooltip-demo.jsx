import { Tooltip, Button, Input } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex flex-col gap-4">
    <Tooltip content="悬停触发的提示">
      <Button>悬停显示</Button>
    </Tooltip>

    <Tooltip content="点击触发的提示" trigger="click">
      <Button>点击显示</Button>
    </Tooltip>

    <Tooltip content="右键触发的提示" trigger="contextMenu">
      <Button>右键显示</Button>
    </Tooltip>

    <Tooltip content="聚焦触发的提示" trigger="focus">
      <Input placeholder="聚焦显示" style={{ width: 200 }} />
    </Tooltip>
  </div>
);

export default Demo;