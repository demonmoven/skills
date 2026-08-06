import { Menu, SplitButton } from '@coze-arch/coze-design';
import { IconCozChatPeople } from '@coze-arch/coze-design/icons';

const Demo = () => (
  <Menu
    trigger="click"
    position="bottomRight"
    className="w-160px"
    render={
      <Menu.SubMenu mode="menu">
        <Menu.Item type="danger" itemKey="测试文字1">
          测试文字1
        </Menu.Item>
        <Menu.Item itemKey="disabled" disabled>
          disabled
        </Menu.Item>
        <Menu.Item itemKey="测试文字2">测试文字2</Menu.Item>
        <Menu.Item itemKey="测试文字3" type="danger" disabled={true}>
          测试文字3
        </Menu.Item>
      </Menu.SubMenu>
    }
  >
    <SplitButton color="highlight" icon={<IconCozChatPeople />}>
      点击菜单
    </SplitButton>
  </Menu>
);

export default Demo;