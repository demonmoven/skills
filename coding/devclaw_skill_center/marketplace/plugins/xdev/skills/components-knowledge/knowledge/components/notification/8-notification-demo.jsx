import React from 'react';
import { Notification, Button } from '@coze-arch/coze-design';

const Demo = () => (
    <Button
        onClick={() => {
            const id = Notification.open({
                title: 'Hi, Bytedance',
                content: 'ies dance dance dance',
                duration: 3,
            })
            setTimeout(() => {
                Notification.open({
                    title: 'Hi, Bytedance',
                    content: 'updated',
                    duration: 10,
                    id
                })
            }, 1000)
        }
        }
    >
        Display Notification
    </Button>
);

export default Demo;
