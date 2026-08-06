import React from 'react';
import { BackTop } from '@coze-arch/coze-design';
import { IconCozLongArrowUp } from '@coze-arch/coze-design/icons';

const Demo = () => {
  const style = {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    height: 30,
    width: 30,
    borderRadius: '100%',
    backgroundColor: '#0077fa',
    color: '#fff',
    bottom: 100,
  };

  return (
    <div>
      <span>
        Scroll down to see the bottom-right <span style={{ color: '#0077fa' }}>blue circular</span> button.
      </span>
      <BackTop style={style}>
        <IconCozLongArrowUp />
      </BackTop>
    </div>
  );
};

export default Demo;
