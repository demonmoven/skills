import React from 'react';
import { Image } from '@coze-arch/coze-design';
import { IconCozWarningCircle } from '@coze-arch/coze-design/icons';

const Demo = () => (
  <div
    style={{ display: 'flex', alignItem: 'center', flexDirection: 'column' }}
  >
    <span>加载失败默认样式</span>
    <Image width={200} height={200} src="https://load-error.jpeg" />
    <br />
    <span>自定义加载失败占位图</span>
    <Image
      width={200}
      height={200}
      src="https://load-error.jpeg"
      fallback={<IconCozWarningCircle style={{ fontSize: 50 }} />}
    />
  </div>
);

export default Demo;
