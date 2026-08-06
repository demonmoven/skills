import React, { useState } from 'react';
import { Spin, Button } from '@coze-arch/coze-design';

const Demo = () => {
    const [loading, toggleLoading] = useState(false);

    const toggle = () => {
        toggleLoading(!loading);
    };
    return (
        <div>
            <Button onClick={toggle} style={{ marginRight: 20 }}>
                延迟显示的spin
            </Button>
            <Spin delay={1000} spinning={loading}></Spin>
        </div>
    );
};

export default Demo;
