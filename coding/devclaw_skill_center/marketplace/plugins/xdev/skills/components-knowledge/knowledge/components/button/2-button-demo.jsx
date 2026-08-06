import { IconButton } from '@coze-arch/coze-design';
import { IconCozPeopleFill, IconCozEdit } from '@coze-arch/coze-design/icons';

const Demo = () => (
  <div style={{ display: 'flex', gap: 8 }}>
    <IconButton icon={<IconCozPeopleFill />} />
    <IconButton icon={<IconCozEdit />} color="highlight" />
    <IconButton icon={<IconCozPeopleFill />} disabled />
  </div>
);

export default Demo;