import { Collapse } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <Collapse accordion>
      <Collapse.Panel header="第一个面板" itemKey="1">
        第一个面板的内容
      </Collapse.Panel>
      <Collapse.Panel header="第二个面板" itemKey="2">
        第二个面板的内容
      </Collapse.Panel>
      <Collapse.Panel header="第三个面板" itemKey="3">
        第三个面板的内容
      </Collapse.Panel>
    </Collapse>
  );
};

export default Demo;