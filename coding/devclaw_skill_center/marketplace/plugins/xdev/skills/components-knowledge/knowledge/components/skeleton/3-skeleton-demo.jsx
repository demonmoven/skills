import React from 'react';
import { Skeleton, Descriptions } from '@coze-arch/coze-design';

const Demo = () => {
    const placeholder = (
        <div>
            <Skeleton.Paragraph rows={1} style={{ width: 80, marginBottom: 10 }} />
            <Skeleton.Title style={{ width: 120 }} />
        </div>
    );

    const data = [{ key: '实际用户数量', value: '1,480,000' }];

    return (
        <Skeleton placeholder={placeholder} loading={true}>
            <Descriptions data={data} row />
        </Skeleton>
    );
};

export default Demo;
