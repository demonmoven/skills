import { Cascader } from '@coze-arch/coze-design';

const treeData = [
  {
    label: '全部类型',
    value: 'all_types',
  },
  {
    label: '工作流',
    value: 'workflow',
  },
  {
    label: '知识库',
    value: 'knowledge',
    children: [
      {
        label: '全部类型',
        value: 'knowledge_all_types',
      },
      {
        label: '文本',
        value: 'text',
      },
      {
        label: '表格',
        value: 'table',
      },
    ],
  },
];

const Demo = () => {
  return (
    <div className="flex flex-col gap-4">
      <Cascader
        placeholder="请选择"
        treeData={treeData}
        style={{ width: '200px' }}
        hasError
      />
    </div>
  );
};

export default Demo;