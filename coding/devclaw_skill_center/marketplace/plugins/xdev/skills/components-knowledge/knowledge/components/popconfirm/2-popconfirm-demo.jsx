import { Popconfirm, Button } from '@coze-arch/coze-design';

const Demo = () => (
  <Popconfirm
    title="确定是否要保存此修改？"
    content="此修改将不可逆"
    cancelText="取消"
  >
    <Button>带取消按钮</Button>
  </Popconfirm>
);

export default Demo;