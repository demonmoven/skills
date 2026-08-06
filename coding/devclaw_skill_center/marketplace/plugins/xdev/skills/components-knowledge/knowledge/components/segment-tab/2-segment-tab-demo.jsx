import { SegmentTab, Space } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
    <div>
      <h4>默认尺寸</h4>
      <SegmentTab
        defaultValue="Tab1"
        options={['Tab1', 'Tab2', 'Tab3']}
        size="default"
      />
    </div>
    <div>
      <h4>小尺寸</h4>
      <SegmentTab
        defaultValue="Tab1"
        options={['Tab1', 'Tab2', 'Tab3']}
        size="small"
      />
    </div>
  </div>
);

export default Demo;