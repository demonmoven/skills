import React from 'react';
import { Notification, Button, ButtonGroup } from '@coze-arch/coze-design';

const Demo = () => {
    let opts = {
        duration: 3,
        position: 'topRight',
        content: 'semi-ui-notification',
        title: 'Hi bytedance',
    };

    return (
        <>
            <ButtonGroup>
                <Button onClick={() => Notification.info({ ...opts, position: 'top' })}>top</Button>
                <Button onClick={() => Notification.info({ ...opts, position: 'topLeft' })}>topLeft</Button>
                <Button onClick={() => Notification.info(opts)}>topRight</Button>
            </ButtonGroup>
            <br />
            <br />
            <ButtonGroup>
                <Button onClick={() => Notification.info({ ...opts, position: 'bottom' })}>bottom</Button>
                <Button onClick={() => Notification.info({ ...opts, position: 'bottomRight' })}>bottomRight</Button>
                <Button onClick={() => Notification.info({ ...opts, position: 'bottomLeft' })}>bottomLeft</Button>
            </ButtonGroup>
        </>
    );
};

export default Demo;
