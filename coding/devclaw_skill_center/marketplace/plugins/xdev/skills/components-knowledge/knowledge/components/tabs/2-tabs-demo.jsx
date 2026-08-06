import React from 'react';
import { Tabs, TabPane } from '@coze-arch/coze-design';

const Demo = () => (
    <Tabs type="button">
        <TabPane tab="文档" itemKey="1">
            文档
        </TabPane>
        <TabPane tab="快速起步" itemKey="2">
            快速起步
        </TabPane>
        <TabPane tab="帮助" itemKey="3">
            帮助
        </TabPane>
    </Tabs>
);

export default Demo;
