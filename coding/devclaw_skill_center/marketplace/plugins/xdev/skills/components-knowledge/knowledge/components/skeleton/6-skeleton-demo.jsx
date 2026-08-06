import React from 'react';
import { Skeleton, Avatar } from '@coze-arch/coze-design';

const Demo = () => {
    const style = {
        display: 'flex',
        alignItems: 'flex-start',
    };

    const placeholder = (
        <div style={style}>
            <Skeleton.Avatar style={{ marginRight: 12 }} />
            <div>
                <Skeleton.Title style={{ width: 120, marginBottom: 12, marginTop: 12 }} />
                <Skeleton.Paragraph style={{ width: 240 }} rows={3} />
            </div>
        </div>
    );

    return (
        <Skeleton placeholder={placeholder} loading={true}>
            <div style={style}>
                <Avatar color="blue" style={{ marginRight: 12 }}>
                    UI
                </Avatar>
                <div>
                    <h3>Semi UI</h3>
                    <p>Hi, Bytedance dance dance.</p>
                    <p>Hi, Bytedance dance dance.</p>
                    <p>Hi, Bytedance dance dance.</p>
                </div>
            </div>
        </Skeleton>
    );
};

export default Demo;
