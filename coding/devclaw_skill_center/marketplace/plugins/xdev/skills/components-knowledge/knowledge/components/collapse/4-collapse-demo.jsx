import { Collapse } from '@coze-arch/coze-design';
import { IconCozArrowDown } from '@coze-arch/coze-design/icons';

const Demo = () => {
  const customHeader = (
    <div className="flex items-center gap-2">
      <IconCozArrowDown />
      <span>自定义头部</span>
    </div>
  );

  return (
    <Collapse>
      <Collapse.Panel header={customHeader} itemKey="1">
        <div className="p-4 bg-gray-100 rounded">
          这是一个自定义样式的内容区域
        </div>
      </Collapse.Panel>
    </Collapse>
  );
};

export default Demo;