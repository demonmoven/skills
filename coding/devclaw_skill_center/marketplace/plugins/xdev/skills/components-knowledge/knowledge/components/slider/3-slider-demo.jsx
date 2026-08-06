import React from 'react';
import { Slider } from '@coze-arch/coze-design';

const Demo = () => (
    <div>
        <Slider tipFormatter={v => (`${v}%`)} getAriaValueText={v => (`${v}%`)}/>
        <br/>
        <br/>
        <Slider tipFormatter={null} />
    </div>
);

export default Demo;
