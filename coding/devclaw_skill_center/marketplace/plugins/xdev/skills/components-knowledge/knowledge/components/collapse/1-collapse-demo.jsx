import { Collapse } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <Collapse>
      <Collapse.Panel header="这是一个折叠面板" itemKey="1">
        这里是折叠面板的内容区域，可以放置任何内容。
      </Collapse.Panel>
      <Collapse.Panel header="这是第二个折叠面板" itemKey="2">
        这里是第二个折叠面板的内容。
      </Collapse.Panel>
    </Collapse>
  );
};

export default Demo;