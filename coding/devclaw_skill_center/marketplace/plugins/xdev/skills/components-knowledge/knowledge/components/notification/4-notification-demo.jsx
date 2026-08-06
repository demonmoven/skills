import React from 'react';
import { Notification, Button } from '@coze-arch/coze-design';

const Demo = () => {
    let opts = {
        title: 'Hi, Bytedance',
        content: 'Hi, Bytedance dance dance',
        duration: 3,
        theme: 'light',
    };

    return (
        <>
            <Button onClick={() => Notification.info(opts)}>Info</Button>
            <br />
            <br />
            <Button onClick={() => Notification.success(opts)}>Success</Button>
            <br />
            <br />
            <Button type="warning" onClick={() => Notification.warning(opts)}>
                Warning
            </Button>
            <br />
            <br />
            <Button type="danger" onClick={() => Notification.error(opts)}>
                Error
            </Button>
        </>
    );
};

export default Demo;
