import React from 'react';
import { Notification, Button } from '@coze-arch/coze-design';

const Demo = () => (
    <Button
        onClick={() =>
            Notification.open({
                title: 'Hi, Bytedance',
                content: 'ies dance dance dance',
                duration: 3,
            })
        }
    >
        Display Notification
    </Button>
);

export default Demo;
