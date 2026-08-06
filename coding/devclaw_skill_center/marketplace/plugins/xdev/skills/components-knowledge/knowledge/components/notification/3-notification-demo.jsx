import React from 'react';
import { Notification, Button } from '@coze-arch/coze-design';
import { IconToutiaoLogo, IconVigoLogo } from '@coze-arch/coze-design/icons';

const Demo = () => {
    let opts = {
        title: 'Hi, Bytedance',
        content: 'ies dance dance dance',
        duration: 3,
    };

    return (
        <>
            <h5>默认的图标</h5>
            <Button type="primary" onClick={() => Notification.success(opts)} style={{ margin: 4 }}>
                Success
            </Button>
            <Button onClick={() => Notification.info(opts)} style={{ margin: 4 }}>
                Info
            </Button>
            <Button type="warning" onClick={() => Notification.warning(opts)} style={{ margin: 4 }}>
                Warning
            </Button>
            <Button type="danger" onClick={() => Notification.error(opts)} style={{ margin: 4 }}>
                Error
            </Button>
            <h5>自定义图标</h5>
            <Button
                icon={<IconToutiaoLogo />}
                style={{ marginRight: 5 }}
                onClick={() =>
                    Notification.info({
                        ...opts,
                        icon: <IconToutiaoLogo style={{ color: 'red' }} />,
                    })
                }
            ></Button>
            <Button
                icon={<IconVigoLogo />}
                style={{ marginRight: 5 }}
                onClick={() => Notification.info({ ...opts, icon: <IconVigoLogo /> })}
            ></Button>
            <Button
                icon={<IconVigoLogo />}
                onClick={() => Notification.info({ ...opts, icon: <IconVigoLogo style={{ color: 'pink' }} /> })}
            ></Button>
        </>
    );
};

export default Demo;
