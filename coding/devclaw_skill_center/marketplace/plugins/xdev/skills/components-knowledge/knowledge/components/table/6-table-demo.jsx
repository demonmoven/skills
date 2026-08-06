import { Table } from '@coze-arch/coze-design';
import { EmptyState } from '@coze-arch/coze-design';

const Demo = () => {
  const columns = [
    {
      title: '名称',
      dataIndex: 'name',
    },
    {
      title: '大小',
      dataIndex: 'size',
    },
    {
      title: '所有者',
      dataIndex: 'owner',
    },
  ];

  return (
    <Table
      tableProps={{
        columns,
        dataSource: [],
      }}
      empty={
        <EmptyState
          title="暂无数据"
          description="当前没有任何数据，请稍后再试"
          image="https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/empty-state/empty_state_no_content.svg"
        />
      }
    />
  );
};

export default Demo;