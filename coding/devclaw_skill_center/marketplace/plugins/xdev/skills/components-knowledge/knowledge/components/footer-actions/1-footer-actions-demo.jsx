import React from 'react';
import { FooterActions } from '@cozeloop/components';

const Demo = () => (
  <FooterActions
    confirmBtnProps={{
      text: '确认',
      onClick: () => alert('确认'),
    }}
    cancelBtnProps={{
      text: '取消',
      onClick: () => alert('取消'),
    }}
  />
);

export default Demo;
