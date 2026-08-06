import React from 'react';
import { Space, Button, Switch } from '@coze-arch/coze-design';

const Demo = () => (
    <Space>
        <Switch defaultChecked={true}/>     
        <Button type="secondary">次要</Button>
        <Button type="tertiary">第三</Button>
        <Button type="warning">警告</Button>
    </Space>
);

export default Demo;
