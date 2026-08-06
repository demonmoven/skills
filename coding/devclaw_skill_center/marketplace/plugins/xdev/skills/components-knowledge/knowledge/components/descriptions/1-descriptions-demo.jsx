import React from 'react';
import { Descriptions, Tag } from '@coze-arch/coze-design';
import { IconCozArrowUp } from '@coze-arch/coze-design/icons';

const Demo = () => {
    const data = [
        { key: '实际用户数量', value: '1,480,000' },
        { key: '7天留存', value: <div>98%<IconCozArrowUp size="small" style={{ color: 'var(--semi-color-success)', marginLeft: '2px' }} /></div> },
        { key: '安全等级', value: '3级' },
        { key: '垂类标签', value: <Tag style={{ margin: 0 }}>电商</Tag> },
        { key: '认证状态', value: '未认证' },
    ];
    return <Descriptions data={data} />;
};

export default Demo;
