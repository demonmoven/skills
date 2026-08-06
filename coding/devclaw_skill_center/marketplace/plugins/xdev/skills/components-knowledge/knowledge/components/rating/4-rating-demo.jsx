import React from 'react';
import { Rating } from '@coze-arch/coze-design';

const Demo = () => (
    <div>
        <span>允许再次点击清除</span>
        <br/>
        <Rating allowClear={true} defaultValue={3}/>
        <br/>
        <br/>
        <span>禁止再次点击清除</span>
        <br/>
        <Rating allowClear={false} defaultValue={3}/>
    </div>
);

export default Demo;
