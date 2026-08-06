import { TabBar } from '@coze-arch/coze-design';

const { TabPanel } = TabBar;

const Demo = () => (
  <div className="flex flex-col gap-8">
    <div>
      <h4>按钮类型（button）</h4>
      <TabBar type="button" defaultActiveKey="1">
        <TabPanel tab="我的" itemKey="1">
          我的内容
        </TabPanel>
        <TabPanel tab="插件" itemKey="2">
          插件内容
        </TabPanel>
        <TabPanel tab="工作流" itemKey="3">
          工作流内容
        </TabPanel>
      </TabBar>
    </div>
    <div>
      <h4>文本类型（text）</h4>
      <TabBar type="text" defaultActiveKey="1">
        <TabPanel tab="我的" itemKey="1">
          我的内容
        </TabPanel>
        <TabPanel tab="插件" itemKey="2">
          插件内容
        </TabPanel>
        <TabPanel tab="工作流" itemKey="3">
          工作流内容
        </TabPanel>
      </TabBar>
    </div>
  </div>
);

export default Demo;