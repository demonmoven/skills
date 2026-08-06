import { Menu, Button } from '@coze-arch/coze-design';
import {
  IconCozArrowDown,
  IconCozCheckMarkFill,
} from '@coze-arch/coze-design/icons';
import { useState } from 'react';

const Demo = () => {
  const [selectedKeys, setSelectedKeys] = useState([]);

  return (
    <div>
      <Menu
        className="w-160px"
        trigger="click"
        position="bottomLeft"
        render={
          <Menu.SubMenu
            mode="selection"
            selectedKeys={selectedKeys}
            onSelectionChange={(value, values) => {
              setSelectedKeys(values);
            }}
          >
            <Menu.Item
              itemKey="测试文字1"
              icon={<IconCozCheckMarkFill className="fill-brand-5 text-lg" />}
            >
              测试文字1
            </Menu.Item>
            <Menu.Item
              itemKey="测试文字2"
              icon={<IconCozCheckMarkFill className="fill-brand-5 text-lg" />}
            >
              测试文字2
            </Menu.Item>
            <Menu.Item
              itemKey="测试文字3"
              icon={<IconCozCheckMarkFill className="fill-brand-5 text-lg" />}
            >
              测试文字3
            </Menu.Item>
          </Menu.SubMenu>
        }
      >
        <Button
          iconPosition="right"
          icon={<IconCozArrowDown className="coz-fg-hglt-plus text-xxl" />}
        >
          单选菜单
        </Button>
      </Menu>
      <div className="text-lg mt-2 text-foreground-4">
        当前选中：{JSON.stringify(selectedKeys)}
      </div>
    </div>
  );
};

export default Demo;