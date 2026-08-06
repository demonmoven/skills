import { Search } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
    <div>
      <h4>默认尺寸</h4>
      <Search size="default" placeholder="默认尺寸" />
    </div>
    <div>
      <h4>小尺寸</h4>
      <Search size="small" placeholder="小尺寸" />
    </div>
  </div>
);

export default Demo;