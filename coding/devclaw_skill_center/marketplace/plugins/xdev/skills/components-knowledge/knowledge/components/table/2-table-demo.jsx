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
  ];

  const data = [
    {
      key: '1',
      name: 'Semi Design 设计稿.fig',
      size: '2M',
      owner: '姜鹏志',
    },
    {
      key: '2',
      name: 'Semi Design 分享演示文稿',
      size: '2M',
      owner: '郝宣',
    },
    {
      key: '3',
      name: '设计文档',
      size: '34KB',
      owner: 'Zoey Edwards',
    },
  ];

  return (
    <Table
      tableProps={{
        columns,
        dataSource: data,
        rowSelection: {
          onChange: (selectedRowKeys, selectedRows) => {
            console.log(
              `selectedRowKeys: ${selectedRowKeys}`,
              'selectedRows: ',
              selectedRows,
            );
          },
        },
      }}
      indexRowSelection={true}
    />
  );
};

export default Demo;