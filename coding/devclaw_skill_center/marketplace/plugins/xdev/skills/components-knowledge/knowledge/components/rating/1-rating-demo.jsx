import React from 'react';
import { Rating } from '@coze-arch/coze-design';

const Demo = () => (
    <div>
        <Rating defaultValue={5}/>
        <br/>
        <br/>
        <Rating size='small' defaultValue={5}/>
    </div>
);

export default Demo;
