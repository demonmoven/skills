import { Table } from '@coze-arch/coze-design';
import { IconCozPeopleFill } from '@coze-arch/coze-design/icons';

const Demo = () => {
  const columns = [
    {
      title: '名称',
      dataIndex: 'name',
      render: (text, record) => (
        <Table.TableMeta
          name={record.name}
          description={record.description}
          icon={<IconCozPeopleFill style={{ fontSize: 24, marginRight: 8 }} />}
        />
      ),
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

  const data = [
    {
      key: '1',
      name: 'Semi Design 设计稿',
      description: '包含所有设计资源和规范',
      size: '2M',
      owner: '姜鹏志',
    },
    {
      key: '2',
      name: 'Semi Design 分享演示文稿',
      description: '分享会议使用的演示文稿',
      size: '2M',
      owner: '郝宣',
    },
    {
      key: '3',
      name: '设计文档',
      description: '详细的设计文档说明',
      size: '34KB',
      owner: 'Zoey Edwards',
    },
  ];

  return (
    <Table
      tableProps={{
        columns,
        dataSource: data,
      }}
    />
  );
};

export default Demo;