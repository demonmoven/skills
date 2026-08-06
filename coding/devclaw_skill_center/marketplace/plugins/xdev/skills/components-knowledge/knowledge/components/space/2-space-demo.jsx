import React from 'react';
import { Space, Button, Tag } from '@coze-arch/coze-design';

const Demo = () => {
    const divStyle = {
        width: 80,
        height: 100,
        lineHight: 100,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        border: '1px solid var(--semi-color-border)',
        borderRadius: 3
    };
    return (
        <Space vertical>
            <Space align='start'>
                <div style={divStyle}>文本</div>
                <Button theme='solid' type='primary'>按钮</Button>
                <Tag color='green' size='large'>标签</Tag>
            </Space>
            <Space align='center'>
                <div style={divStyle}>文本</div>
                <Button theme='solid' type='primary'>按钮</Button>
                <Tag color='green' size='large'>标签</Tag>
            </Space>
            <Space align='end'>
                <div style={divStyle}>文本</div>
                <Button theme='solid' type='primary'>按钮</Button>
                <Tag color='green' size='large'>标签</Tag>
            </Space>
            <Space align='baseline'>
                <div style={divStyle}>文本</div>
                <Button theme='solid' type='primary'>按钮</Button>
                <Tag color='green' size='large'>标签</Tag>
            </Space>
        </Space>
    );
};

export default Demo;
