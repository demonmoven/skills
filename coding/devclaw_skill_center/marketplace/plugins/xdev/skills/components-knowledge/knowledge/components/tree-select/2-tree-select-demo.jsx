import { TreeSelect } from '@coze-arch/coze-design';

const treeData = [
  {
    label: '亚洲',
    value: 'Asia',
    key: '0',
    children: [
      {
        label: '中国',
        value: 'China',
        key: '0-0',
        children: [
          {
            label: '北京',
            value: 'Beijing',
            key: '0-0-0',
          },
          {
            label: '上海',
            value: 'Shanghai',
            key: '0-0-1',
          },
        ],
      },
    ],
  },
  {
    label: '北美洲',
    value: 'North America',
    key: '1',
  },
];

const Demo = () => (
  <TreeSelect
    multiple
    leafOnly
    treeData={treeData}
    placeholder="请选择"
    style={{ width: 240 }}
    dropdownStyle={{ maxHeight: 400, overflow: 'auto' }}
  />
);

export default Demo;