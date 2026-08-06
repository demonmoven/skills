import { AIButton } from '@coze-arch/coze-design';
import { IconCozPeopleFill } from '@coze-arch/coze-design/icons';

const Demo = () => (
  <div style={{ display: 'flex', gap: 8 }}>
    <AIButton color="aihglt">aihglt</AIButton>
    <AIButton color="aiplus">AI Plus</AIButton>
    <AIButton color="aiprimary">AI Primary</AIButton>

    <AIButton color="aiplus" icon={<IconCozPeopleFill />}>
      AI Plus
    </AIButton>

    <AIButton color="aiplus" hideIcon={true}>
      AI Plus
    </AIButton>

    <AIButton color="aiplus" loading>
      AI Plus
    </AIButton>
  </div>
);

export default Demo;