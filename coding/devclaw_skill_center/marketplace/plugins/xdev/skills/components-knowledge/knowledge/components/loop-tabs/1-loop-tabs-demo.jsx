import React from 'react';
import { LoopTabs } from '@cozeloop/components';
import { TabPane } from '@coze-arch/coze-design';

const Demo = () => (
  <LoopTabs defaultActiveKey="1">
    <TabPane tab="标签一" itemKey="1">
      <div>标签一内容</div>
    </TabPane>
    <TabPane tab="标签二" itemKey="2">
      <div>标签二内容</div>
    </TabPane>
    <TabPane tab="标签三" itemKey="3">
      <div>标签三内容</div>
    </TabPane>
  </LoopTabs>
);

export default Demo;
