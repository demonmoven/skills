import React from 'react';
import { Rating } from '@coze-arch/coze-design';

const Demo = () => (
    <div>
        <Rating allowHalf defaultValue={3.5}/>
        <br/>
        <Rating allowHalf defaultValue={3.65} disabled/>
    </div>
);

export default Demo;
