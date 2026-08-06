// NOTE: 以下仅为文档演示，实际使用时请使用正确的引入方式
// import { Menu, Button } from '@coze-arch/coze-design';
import { Menu, Button } from '@coze-arch/coze-design';

const data = [
  { node: 'title', name: '分组1' },
  {
    node: 'item',
    name: '分组内容',
    type: 'primary',
    onClick: () => console.log('click primary'),
  },
  { node: 'item', name: 'secondary', type: 'secondary' },
  { node: 'divider' },
  { node: 'title', name: '分组2' },
  { node: 'item', name: 'tertiary', type: 'tertiary' },
  { node: 'item', name: 'warning', type: 'warning', active: true },
  { node: 'item', name: 'danger', type: 'danger' },
];

const Demo = () => (
  <Menu
    trigger="click"
    showTick
    position="bottomLeft"
    menu={data}
    className="w-160px"
  >
    <Button theme="outline" type="tertiary">
      Click Me
    </Button>
  </Menu>
);

export default Demo;