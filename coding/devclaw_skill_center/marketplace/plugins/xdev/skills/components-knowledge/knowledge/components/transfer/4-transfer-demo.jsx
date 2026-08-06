import React from 'react';
import { Transfer } from '@coze-arch/coze-design';

const Demo = () => {
    const data = Array.from({ length: 20 }, (v, i) => {
        return {
            label: `选项名称 ${i}`,
            value: i,
            disabled: false,
            key: i,
        };
    });
    return (
        <Transfer
            style={{ width: 568, height: 416 }}
            disabled
            dataSource={data}
            defaultValue={[2, 4]}
            onChange={(values, items) => console.log(values, items)}
        />
    );
};

export default Demo;
