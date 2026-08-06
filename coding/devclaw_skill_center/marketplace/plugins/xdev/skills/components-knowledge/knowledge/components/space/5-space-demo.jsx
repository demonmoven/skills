import React from 'react';
import { Space, Button } from '@coze-arch/coze-design';

const Demo = () => (
    <Space wrap>
        {new Array(10).fill(null).map((item, idex) => (
            <Button theme='solid' type='secondary' key={idex}>按钮</Button>
        ))}
    </Space>
);

export default Demo;
