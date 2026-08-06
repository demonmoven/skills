import { Switch } from '@coze-arch/coze-design';
import { useState } from 'react';

const Demo = () => {
  const [checked, setChecked] = useState(false);

  return (
    <div className="flex flex-col gap-2">
      <Switch
        checked={checked}
        onChange={value => {
          console.log('switch changed:', value);
          setChecked(value);
        }}
      />
      <div>当前状态：{checked ? '开启' : '关闭'}</div>
    </div>
  );
};

export default Demo;