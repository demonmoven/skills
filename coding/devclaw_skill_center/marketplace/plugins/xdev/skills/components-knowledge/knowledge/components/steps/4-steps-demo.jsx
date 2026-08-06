import React from 'react';
import { Steps } from '@coze-arch/coze-design';

const Demo = () => (
    <Steps type="basic" size="small" current={1} onChange={(i)=>console.log(i)}>
        <Steps.Step title="Finished" description="This is a description" />
        <Steps.Step title="In Progress" description="This is a description" />
        <Steps.Step title="Waiting" description="This is a description" />
    </Steps>
);

export default Demo;
