import React from 'react';
import { Descriptions } from '@coze-arch/coze-design';
import { IconCozArrowUp } from '@coze-arch/coze-design/icons';

const Demo = () => {
    const data = [
        { key: '实际用户数量', value: '1,480,000' },
        {
            key: '7天留存',
            value: (
                <span>
                    98%
                    <IconCozArrowUp size="small" style={{ color: 'red', marginLeft: '4px' }} />
                </span>
            ),
        },
        { key: '安全等级', value: '3级' },
    ];
    const style = {
        boxShadow: 'var(--semi-shadow-elevated)',
        backgroundColor: 'var(--semi-color-bg-2)',
        borderRadius: '4px',
        padding: '10px',
        marginRight: '20px',
        width: '600px',
    };
    return (
        <div>
            <Descriptions data={data} row size="small" style={style} />
            <br />
            <Descriptions data={data} row style={style} />
            <br />
            <Descriptions data={data} row size="large" style={style} />
        </div>
    );
};

export default Demo;
