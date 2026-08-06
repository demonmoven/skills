import { Radio } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
    <div>
      <h4>水平排列</h4>
      <Radio.Group direction="horizontal" defaultValue={1}>
        <Radio value={1}>选项A</Radio>
        <Radio value={2}>选项B</Radio>
        <Radio value={3}>选项C</Radio>
      </Radio.Group>
    </div>

    <div>
      <h4>垂直排列</h4>
      <Radio.Group direction="vertical" defaultValue={1}>
        <Radio value={1}>选项A</Radio>
        <Radio value={2}>选项B</Radio>
        <Radio value={3}>选项C</Radio>
      </Radio.Group>
    </div>

    <div>
      <h4>禁用状态</h4>
      <Radio.Group disabled direction="horizontal" defaultValue={1}>
        <Radio value={1}>选项A</Radio>
        <Radio value={2}>选项B</Radio>
        <Radio value={3}>选项C</Radio>
      </Radio.Group>
    </div>
  </div>
);

export default Demo;