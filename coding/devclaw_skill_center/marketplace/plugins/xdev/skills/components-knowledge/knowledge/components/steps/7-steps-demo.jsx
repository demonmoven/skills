import React from 'react';
import { Steps } from '@coze-arch/coze-design';

const Demo = () => (
    <Steps direction="vertical" current={1} style={{ width: 300 }} onChange={(i)=>console.log(i)}>
        <Steps.Step title="Finished" description="This is a description" />
        <Steps.Step title="In Progress" description="This is a description" />
        <Steps.Step title="Waiting" description="This is a description" />
    </Steps>
);

export default Demo;
