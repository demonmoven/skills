import React, { useState } from 'react';
import { StepNav } from '@cozeloop/components';

const Demo = () => {
  const [currentStep, setCurrentStep] = useState('step1');

  return (
    <StepNav
      currentStep={currentStep}
      onStepChange={setCurrentStep}
      clickToChange
      stepItems={[
        { key: 'step1', label: '基本信息' },
        { key: 'step2', label: '配置参数' },
        { key: 'step3', label: '确认提交' },
      ]}
    />
  );
};

export default Demo;
