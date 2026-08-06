import { Checkbox } from '@coze-arch/coze-design';
import { useState } from 'react';

const Demo = () => {
  const [checked, setChecked] = useState(false);

  return (
    <Checkbox checked={checked} onChange={e => setChecked(e.target.checked)}>
      受控复选框
    </Checkbox>
  );
};

export default Demo;