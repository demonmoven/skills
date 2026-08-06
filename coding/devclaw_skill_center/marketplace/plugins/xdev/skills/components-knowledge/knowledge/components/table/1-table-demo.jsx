import { Table } from '@coze-arch/coze-design';

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
    {
      title: '更新日期',
      dataIndex: 'updateTime',
    },
  ];

  const data = [
    {
      key: '1',
      name: 'Semi Design 设计稿.fig',
      size: '2M',
      owner: '姜鹏志',
      updateTime: '2020-02-02 05:13',
    },
    {
      key: '2',
      name: 'Semi Design 分享演示文稿',
      size: '2M',
      owner: '郝宣',
      updateTime: '2020-01-17 05:31',
    },
    {
      key: '3',
      name: '设计文档',
      size: '34KB',
      owner: 'Zoey Edwards',
      updateTime: '2020-01-26 11:01',
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