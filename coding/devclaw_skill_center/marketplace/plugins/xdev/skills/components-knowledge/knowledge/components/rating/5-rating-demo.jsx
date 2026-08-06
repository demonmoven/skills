import React, { useState } from 'react';
import { Rating } from '@coze-arch/coze-design';

const Demo = () => {
    const [value, setValue] = useState(0);
    const change = (val) => setValue(val);
    const desc = ['terrible', 'bad', 'normal', 'good', 'wonderful'];
    return (
        <div>
            <span>How was the help you received: 
                {value ? <span>{desc[value - 1]}</span> : ''}
            </span>
            <br/>
            <Rating tooltips={desc} onChange={change} value={value} />
        </div>
    );
};

export default Demo;
