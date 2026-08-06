import { IconCozPeopleFill } from '@coze-arch/coze-design/icons';

const Demo = () => (
  <div style={{ display: 'flex', gap: 16, alignItems: 'center' }}>
    <IconCozPeopleFill className="text-xxl" /> {/* 16px */}
    <IconCozPeopleFill className="text-lg" /> {/* 14px */}
    <IconCozPeopleFill className="text-base" /> {/* 12px */}
  </div>
);

export default Demo;