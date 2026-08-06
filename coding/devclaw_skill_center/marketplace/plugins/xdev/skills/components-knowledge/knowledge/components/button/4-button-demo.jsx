import { SplitButton } from '@coze-arch/coze-design';
import { IconCozArrowDown } from '@coze-arch/coze-design/icons';

const Demo = () => (
  <div style={{ display: 'flex', gap: 8 }}>
    <SplitButton icon={<IconCozArrowDown />}>分裂按钮</SplitButton>
    <SplitButton color="highlight" icon={<IconCozArrowDown />}>
      高亮分裂按钮
    </SplitButton>
  </div>
);

export default Demo;