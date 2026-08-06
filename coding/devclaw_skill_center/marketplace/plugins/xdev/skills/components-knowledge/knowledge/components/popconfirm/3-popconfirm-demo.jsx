import { Popconfirm, Button } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="space-x-4">
    <Popconfirm
      title="确定是否要保存此修改？"
      content="此修改将不可逆"
      okButtonColor="brand"
    >
      <Button color="brand">品牌色</Button>
    </Popconfirm>

    <Popconfirm
      title="确定是否要删除此项？"
      content="删除后将无法恢复"
      okButtonColor="yellow"
    >
      <Button color="yellow">警告</Button>
    </Popconfirm>

    <Popconfirm
      title="确定要执行此危险操作？"
      content="此操作可能造成不可逆的影响"
      okButtonColor="red"
    >
      <Button color="red">危险</Button>
    </Popconfirm>
  </div>
);

export default Demo;