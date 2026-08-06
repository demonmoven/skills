import React, { useState } from 'react';
import { SideSheet, RadioGroup, Radio, Button } from '@coze-arch/coze-design';

const Demo = () => {
    const [visible, setVisible] = useState(false);
    const change = () => {
        setVisible(!visible);
    };
    const [size, setSize] = useState('small');
    const changeSize = e => {
        setSize(e.target.value);
    };
    return (
        <>
            <RadioGroup onChange={changeSize} value={size}>
                <Radio value={'small'}>small</Radio>
                <Radio value={'medium'}>medium</Radio>
                <Radio value={'large'}>large</Radio>
            </RadioGroup>
            <br />
            <br />
            <Button onClick={change}>Open SideSheet</Button>
            <SideSheet title="滑动侧边栏" visible={visible} onCancel={change} size={size}>
                <p>This is the content of a basic sidesheet.</p>
                <p>Here is more content...</p>
            </SideSheet>
        </>
    );
};

export default Demo;
