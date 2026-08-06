import { Banner } from '@coze-arch/coze-design';
import { useState } from 'react';

const Demo = () => {
  const [visible, setVisible] = useState(true);

  return visible ? (
    <Banner
      fullMode={true}
      description={
        <div>
          这是一条可关闭的提示信息
          <span
            className="coz-fg-hglt pl-2 cursor-pointer"
            onClick={() => setVisible(false)}
          >
            不再显示
          </span>
        </div>
      }
    />
  ) : null;
};

export default Demo;