import { InputCode } from '@coze-arch/coze-design';
import { useState } from 'react';

const Demo = () => {
  const [value, setValue] = useState('');

  return (
    <div className="flex flex-col gap-4">
      <InputCode
        value={value}
        onChange={setValue}
        onFinish={value => console.log('输入完成：', value)}
      />
      <div>当前输入：{value}</div>
    </div>
  );
};

export default Demo;