import { IconCozPeopleFill, IconCozEdit } from '@coze-arch/coze-design/icons';
import { IconButton } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', gap: 16, alignItems: 'center' }}>
    <IconButton icon={<IconCozPeopleFill className="text-xxl coz-fg-hglt" />} />
    <IconButton icon={<IconCozEdit className="text-xxl coz-fg-hglt-red" />} />
  </div>
);

export default Demo;