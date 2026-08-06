import { Radio } from '@coze-arch/coze-design';

import { useState } from 'react';

const Demo = () => {
  const [value] = useState(1);
  return (
    <div className="flex gap-12">
      <Radio
        mode="advanced"
        type="pureCard"
        checked={value === 1}
        extra="这是一段描述文本，用于解释该选项的更多信息。"
      >
        这是第一个选项
      </Radio>
      <Radio
        mode="advanced"
        type="pureCard"
        checked={value === 2}
        extra="这是一段描述文本，用于解释该选项的更多信息。"
      >
        这是第二个选项
      </Radio>

      <Radio.Group type="pureCard">
        <Radio value={1}>选项A</Radio>
        <Radio value={2}>选项B</Radio>
        <Radio value={3}>选项C</Radio>
      </Radio.Group>
    </div>
  );
};

export default Demo;
