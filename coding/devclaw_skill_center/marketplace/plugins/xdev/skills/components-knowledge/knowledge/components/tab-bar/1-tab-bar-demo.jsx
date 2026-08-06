import { TabBar } from '@coze-arch/coze-design';

const { TabPanel } = TabBar;

const Demo = () => (
  <TabBar defaultActiveKey="1">
    <TabPanel tab="效率工具" itemKey="1">
      效率工具内容
    </TabPanel>
    <TabPanel tab="商务服务" itemKey="2">
      商务服务内容
    </TabPanel>
    <TabPanel tab="文本创作" itemKey="3">
      文本创作内容
    </TabPanel>
    <TabPanel tab="学习教育" itemKey="4">
      学习教育内容
    </TabPanel>
  </TabBar>
);

export default Demo;