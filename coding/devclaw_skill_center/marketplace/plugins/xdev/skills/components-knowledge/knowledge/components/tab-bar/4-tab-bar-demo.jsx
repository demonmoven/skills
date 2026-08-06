import { TabBar } from '@coze-arch/coze-design';

const { TabPanel } = TabBar;

const Demo = () => (
  <div className="flex flex-col gap-8 w-full">
    <div>
      <h4>左对齐（默认）</h4>
      <TabBar align="left" defaultActiveKey="1">
        <TabPanel tab="效率工具" itemKey="1" />
        <TabPanel tab="商务服务" itemKey="2" />
        <TabPanel tab="文本创作" itemKey="3" />
      </TabBar>
    </div>
    <div>
      <h4>居中对齐</h4>
      <TabBar align="center" defaultActiveKey="1">
        <TabPanel tab="效率工具" itemKey="1" />
        <TabPanel tab="商务服务" itemKey="2" />
        <TabPanel tab="文本创作" itemKey="3" />
      </TabBar>
    </div>
    <div>
      <h4>右对齐</h4>
      <TabBar align="right" defaultActiveKey="1">
        <TabPanel tab="效率工具" itemKey="1" />
        <TabPanel tab="商务服务" itemKey="2" />
        <TabPanel tab="文本创作" itemKey="3" />
      </TabBar>
    </div>
  </div>
);

export default Demo;