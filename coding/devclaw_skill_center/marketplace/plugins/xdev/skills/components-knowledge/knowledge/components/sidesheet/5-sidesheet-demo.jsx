import React, { useState } from 'react';
import { SideSheet, Button } from '@coze-arch/coze-design';

const Demo = () => {
    const [visible, setVisible] = useState(false);
    const [value, setValue] = useState('');
    const getContainer = () => {
        return document.querySelector('.sidesheet-container');
    };
    return (
        <div
            style={{
                height: 320,
                overflow: 'hidden',
                position: 'relative',
                border: '1px solid var(--semi-color-border)',
                borderRadius: 2,
                padding: 24,
                textAlign: 'center',
                background: 'var(--semi-color-fill-0)',
            }}
            className="sidesheet-container"
        >
            <span>Render in this</span>
            <br />
            <br />
            <Button onClick={() => setVisible(true)}>Open SideSheet</Button>
            <SideSheet
                title="渲染在指定容器内部"
                visible={visible}
                onCancel={() => setVisible(false)}
                width={220}
                getPopupContainer={getContainer}
            >
                <p>This is the content of a basic sidesheet.</p>
                <p>Here is more content...</p>
            </SideSheet>
        </div>
    );
};

export default Demo;
