import React from 'react';
import { Slider } from '@coze-arch/coze-design';

const Demo = () => (
  <div>
        <div>
            <div>Default</div>
            <Slider showBoundary={true} handleDot={{size:'4px',color:'blue'}}></Slider>
        </div>
        <br/>
        <br/>
        <div>
            <div>Range</div>
            <Slider defaultValue={[20, 60]} range handleDot={[{size:'4px',color:'blue'},{size:'4px',color:'pink'}]}></Slider>
        </div>
    </div>
);

export default Demo;
