import {
  IconCozIllusDone,
  IconCozIllusDoneDark,
} from '@coze-arch/coze-design/illustrations';

const Demo = () => (
  <div style={{ display: 'flex', gap: 16, alignItems: 'center' }}>
    <div>
      <p>亮色模式</p>
      <IconCozIllusDone width="120" height="120" />
    </div>
    <div>
      <p>暗色模式</p>
      <IconCozIllusDoneDark width="120" height="120" />
    </div>
  </div>
);

export default Demo;