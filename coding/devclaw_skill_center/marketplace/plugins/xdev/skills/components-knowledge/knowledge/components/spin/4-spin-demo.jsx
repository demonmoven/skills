import React from 'react';
import { Spin } from '@coze-arch/coze-design';
import { IconCozLoading } from '@coze-arch/coze-design/icons';

const Demo = () => (
    <div style={{ marginLeft: 30 }}>
        <div>A spin with customized indicator.</div>
        <Spin indicator={<IconCozLoading />} />
    </div>
);

export default Demo;
