import { TabBar } from '@coze-arch/coze-design';
import { useState } from 'react';

const { TabPanel } = TabBar;

const Demo = () => {
  const [selectedKey, setSelectedKey] = useState('效率工具');

  return (
    <div className="flex flex-col gap-8">
      <div>
        <h4>标签页模式（tab）</h4>
        <TabBar mode="tab" defaultActiveKey="1">
          <TabPanel tab="效率工具" itemKey="1">
            效率工具内容
          </TabPanel>
          <TabPanel tab="商务服务" itemKey="2">
            商务服务内容
          </TabPanel>
          <TabPanel tab="文本创作" itemKey="3">
            文本创作内容
          </TabPanel>
        </TabBar>
      </div>
      <div>
        <h4>选择器模式（select）</h4>
        <div className="flex flex-col gap-2">
          <TabBar
            mode="select"
            activeKey={selectedKey}
            onTabClick={key => setSelectedKey(key)}
          >
            <TabPanel tab="效率工具" itemKey="效率工具" />
            <TabPanel tab="商务服务" itemKey="商务服务" />
            <TabPanel tab="文本创作" itemKey="文本创作" />
          </TabBar>
          <div className="text-lg text-foreground-4">已选中：{selectedKey}</div>
        </div>
      </div>
    </div>
  );
};

export default Demo;