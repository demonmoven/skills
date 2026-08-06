import React, { useState } from 'react';
import { Notification, Button } from '@coze-arch/coze-design';

const Demo = () => {
    let opts = {
        content: 'Not auto close',
        title: 'Hi',
        duration: 0,
    };
    const [ids, setIds] = useState([]);
    function show() {
        let id = Notification.info(opts);
        setIds([...ids, id]);
    }
    function hide() {
        let idsTmp = [...ids];
        Notification.close(idsTmp.shift());
        setIds(idsTmp);
    }
    return (
        <>
            <Button type="primary" onClick={show}>
                Show Notification
            </Button>
            <br />
            <br />
            <Button type="primary" onClick={hide}>
                Hide Notification
            </Button>
        </>
    );
};

export default Demo;
