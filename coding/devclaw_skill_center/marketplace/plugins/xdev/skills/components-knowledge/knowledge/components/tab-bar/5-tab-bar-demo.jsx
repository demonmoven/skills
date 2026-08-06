import { TabBar, Badge } from '@coze-arch/coze-design';

const { TabPanel } = TabBar;

const Demo = () => (
  <div className="p-4">
    <TabBar defaultActiveKey="1">
      <TabPanel tab="效率工具" itemKey="1">
        效率工具内容
      </TabPanel>
      <TabPanel tab="商务服务" itemKey="2">
        商务服务内容
      </TabPanel>
      <TabPanel
        tab={
          <div className="relative">
            <Badge type="success" count="beta" style={{ right: -6, top: -6 }}>
              <span>新功能</span>
            </Badge>
          </div>
        }
        itemKey="3"
      >
        新功能内容
      </TabPanel>
    </TabBar>
  </div>
);

export default Demo;