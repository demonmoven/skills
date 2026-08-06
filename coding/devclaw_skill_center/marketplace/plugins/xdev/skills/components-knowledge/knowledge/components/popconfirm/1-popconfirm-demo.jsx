import { Popconfirm, Button } from '@coze-arch/coze-design';

const Demo = () => (
  <Popconfirm title="确定是否要保存此修改？" content="此修改将不可逆">
    <Button>点击确认</Button>
  </Popconfirm>
);

export default Demo;