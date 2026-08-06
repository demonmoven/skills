import React from 'react';
import { Skeleton, Button } from '@coze-arch/coze-design';

const Demo = () => {
    const style = {
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        width: '300px',
        marginBottom: '10px',
    };

    const placeholder = (
        <div style={style}>
            <Skeleton.Paragraph style={style} rows={3} />
            <Skeleton.Button />
        </div>
    );

    return (
        <Skeleton placeholder={placeholder} loading={true} style={{ textAlign: 'center' }}>
            <div style={{ textAlign: 'center' }}>
                <p>Hi, Bytedance dance dance.</p>
                <p>Hi, Bytedance dance dance.</p>
                <Button>Button</Button>
            </div>
        </Skeleton>
    );
};

export default Demo;
