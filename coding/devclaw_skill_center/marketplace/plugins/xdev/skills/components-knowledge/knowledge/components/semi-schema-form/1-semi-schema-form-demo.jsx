import React, { useRef } from 'react';
import { SemiSchemaForm, schemaValidators } from '@cozeloop/components';

const schema = {
  type: 'object',
  properties: {
    name: {
      type: 'string',
      title: '名称',
    },
    age: {
      type: 'number',
      title: '年龄',
    },
  },
  required: ['name'],
};

const uiSchema = {};

const Demo = () => {
  const formRef = useRef(null);

  return (
    <SemiSchemaForm
      ref={formRef}
      schema={schema}
      uiSchema={uiSchema}
      validator={schemaValidators}
      onSubmit={({ formData }) => console.log('submitted', formData)}
    />
  );
};

export default Demo;
