import { Menu, Button } from '@coze-arch/coze-design';
import {
  IconCozArrowDown,
  IconCozCheckMarkFill,
  IconCozLoading,
} from '@coze-arch/coze-design/icons';
import { useState } from 'react';

const Demo = () => {
  const [selectedKeys, setSelectedKeys] = useState([]);

  return (
    <div>
      <Menu
        trigger="click"
        position="bottomLeft"
        className="w-[220px]"
        render={
          <Menu.SubMenu
            mode="selection"
            multiple={true}
            selectedKeys={selectedKeys}
            onSelectionChange={(value, values) => {
              setSelectedKeys(values);
            }}
          >
            <Menu.Title>分组1</Menu.Title>
            <Menu.Divider />
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
            <Menu.Title>分组2</Menu.Title>
            <Menu.Divider />
            <Menu.Item
              itemKey="测试文字4"
              icon={<IconCozCheckMarkFill className="fill-brand-5 text-lg" />}
              suffix={<span className="text-foreground-3">suffix</span>}
            >
              测试文字4
            </Menu.Item>
            <Menu.Item
              onClick={value => {
                console.log('current:', value);
              }}
              itemKey="测试文字5"
              icon={<IconCozCheckMarkFill className="fill-brand-5 text-lg" />}
              suffix={<IconCozLoading className="fill-red-5 text-lg" />}
            >
              测试文字5
            </Menu.Item>
          </Menu.SubMenu>
        }
      >
        <Button
          iconPosition="right"
          icon={<IconCozArrowDown className="coz-fg-hglt-plus text-xxl" />}
        >
          多选菜单
        </Button>
      </Menu>
      <div className="text-lg mt-2 text-foreground-4">
        当前选中：{JSON.stringify(selectedKeys)}
      </div>
    </div>
  );
};

export default Demo;