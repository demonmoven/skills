import { Radio } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
    <Radio defaultChecked>默认选中</Radio>
    <Radio>未选中</Radio>
    <Radio checked>受控选中</Radio>
    <Radio disabled>禁用未选中</Radio>
    <Radio disabled checked>
      禁用已选中
    </Radio>
  </div>
);

export default Demo;