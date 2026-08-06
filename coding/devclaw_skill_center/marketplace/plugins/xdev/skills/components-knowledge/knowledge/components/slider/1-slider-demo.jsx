import React from 'react';
import { Slider } from '@coze-arch/coze-design';

const Demo = () => (
    <div>
        <div>
            <div>Default</div>
            <Slider showBoundary={true}></Slider>
        </div>
        <br/>
        <br/>
        <div>
            <div>Range</div>
            <Slider defaultValue={[20, 60]} range></Slider>
        </div>
        <br/>
        <br/>
        <div>
            <div>Disabled</div>
            <Slider defaultValue={40} disabled></Slider>
        </div>
    </div>
);

export default Demo;
