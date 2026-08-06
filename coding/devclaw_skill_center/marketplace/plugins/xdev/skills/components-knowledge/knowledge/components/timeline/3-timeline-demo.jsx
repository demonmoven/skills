import React from 'react';
import { Timeline } from '@coze-arch/coze-design';
import { IconCozWarning } from '@coze-arch/coze-design/icons';

const Demo = () => (
    <Timeline>
        <Timeline.Item time="2019-07-14 10:35">默认样式的节点</Timeline.Item>
        <Timeline.Item time="2019-06-13 16:17" dot={<IconCozWarning />} type="warning">
            自定义图标
        </Timeline.Item>
        <Timeline.Item time="2019-05-14 18:34" color="pink">
            自定义节点颜色
        </Timeline.Item>
        <Timeline.Item time="2019-04-10 12:20">
            <span style={{ fontSize: '18px' }}>自定义节点样式</span>
        </Timeline.Item>
    </Timeline>
);

export default Demo;
