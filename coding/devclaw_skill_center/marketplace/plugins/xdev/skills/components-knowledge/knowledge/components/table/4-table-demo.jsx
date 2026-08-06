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
      title: '操作',
      dataIndex: 'operation',
      render: (text, record) => (
        <Table.TableAction
          editProps={{
            handler: () => {
              console.log('编辑', record);
            },
          }}
          copyProps={{
            handler: () => {
              console.log('复制', record);
            },
          }}
          deleteProps={{
            handler: () => {
              console.log('删除', record);
            },
            deleteDesc: `确定要删除 "${record.name}" 吗？`,
          }}
          actionList={[
            {
              actionKey: 'download',
              actionText: '下载',
              handler: () => {
                console.log('下载', record);
              },
            },
          ]}
        />
      ),
    },
  ];

  const data = [
    {
      key: '1',
      name: 'Semi Design 设计稿.fig',
      size: '2M',
    },
    {
      key: '2',
      name: 'Semi Design 分享演示文稿',
      size: '2M',
    },
    {
      key: '3',
      name: '设计文档',
      size: '34KB',
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