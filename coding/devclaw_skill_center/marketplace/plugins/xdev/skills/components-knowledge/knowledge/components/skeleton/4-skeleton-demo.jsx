import React from 'react';
import { Skeleton, Avatar } from '@coze-arch/coze-design';

const Demo = () => {
    const placeholder = (
        <div style={{ display: 'flex', alignItems: 'center' }}>
            <Skeleton.Avatar style={{ marginRight: 12 }} />
            <Skeleton.Title style={{ width: 120 }} />
        </div>
    );

    return (
        <Skeleton placeholder={placeholder} loading={true}>
            <Avatar color="blue" style={{ marginRight: 12 }}>
                UI
            </Avatar>
            <span>Semi UI</span>
        </Skeleton>
    );
};

export default Demo;
