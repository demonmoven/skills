import React, { useRef } from 'react';
import { SentinelForm } from '@cozeloop/components';
import { Form } from '@coze-arch/coze-design';

const Demo = () => {
  const formRef = useRef(null);

  return (
    <SentinelForm
      ref={formRef}
      formID="demo-module-create-form"
      onSubmit={(values) => console.log('submitted', values)}
      onValueChange={(values, changed) => console.log('changed', changed)}
    >
      <Form.Input field="name" label="名称" placeholder="请输入名称" />
      <Form.Input field="description" label="描述" placeholder="请输入描述" />
    </SentinelForm>
  );
};

export default Demo;
