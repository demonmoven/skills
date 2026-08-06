import React, { useState } from 'react';
import { Collapsible, InputNumber, Button } from '@coze-arch/coze-design';

const Demo = () => {
    const [isOpen, setOpen] = useState(false);
    const [duration, setDuration] = useState(250);
    const toggle = () => {
        setOpen(!isOpen);
    };
    const collapsed = (
        <ul>
            <li>
                <p>Semi Design 以内容优先进行设计。</p>
            </li>
            <li>
                <p>更容易地自定义主题。</p>
            </li>
            <li>
                <p>适用国际化场景。</p>
            </li>
            <li>
                <p>效率场景加入人性化关怀。</p>
            </li>
        </ul>
    );
    return (
        <div>
            <label>设置动画时间：</label>
            <InputNumber min={0} defaultValue={250} style={{ width: 120 }} onChange={(val) => setDuration(val)} step={10} />
            <br />
            <Button onClick={toggle}>Toggle</Button>
            <Collapsible isOpen={isOpen} duration={duration}>
                {collapsed}
            </Collapsible>
        </div>
    );
};

export default Demo;
