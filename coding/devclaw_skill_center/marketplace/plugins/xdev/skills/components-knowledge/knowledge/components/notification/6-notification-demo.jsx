import React from 'react';
import { Notification, Button } from '@coze-arch/coze-design';

const Demo = () => {
    let opts = {
        content: 'Hi, Bytedance dance dance',
        duration: 10,
    };

    return <Button onClick={() => Notification.info(opts)}>Close After 10s</Button>;
};

export default Demo;
