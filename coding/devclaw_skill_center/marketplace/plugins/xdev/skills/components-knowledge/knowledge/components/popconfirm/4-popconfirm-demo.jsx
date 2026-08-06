import { Popconfirm, Button } from '@coze-arch/coze-design';

const Demo = () => (
  <Popconfirm
    title="确定要删除此文件？"
    content={
      <>
        <Popconfirm.SubTitle>重要提示</Popconfirm.SubTitle>
        <Popconfirm.Description>此操作将永久删除该文件</Popconfirm.Description>
        <Popconfirm.SubTitle>影响范围</Popconfirm.SubTitle>
        <Popconfirm.Description>删除后将无法恢复</Popconfirm.Description>
      </>
    }
  >
    <Button>自定义内容</Button>
  </Popconfirm>
);

export default Demo;