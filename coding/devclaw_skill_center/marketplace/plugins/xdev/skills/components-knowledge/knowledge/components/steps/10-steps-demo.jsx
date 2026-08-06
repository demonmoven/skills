import React from 'react';
import { Steps } from '@coze-arch/coze-design';
import { IconCozHouse, IconCozLock } from '@coze-arch/coze-design/icons';

const Demo = () => (
    <Steps type="basic" onChange={(i)=>console.log(i)}>
        <Steps.Step status="finish" title="已完成" />
        <Steps.Step status="error" title="错误" />
        <Steps.Step status="warning" title="警告" />
        <Steps.Step status="process" title="正在进行" icon={<IconCozHouse size="extra-large" />} />
        <Steps.Step status="wait" title="等待" icon={<IconCozLock size="extra-large" />} />
    </Steps>
);

export default Demo;
