import { SingleSelect } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
    <div>
      <h4>Hug 布局（默认）</h4>
      <SingleSelect
        layout="hug"
        defaultValue="1"
        options={[
          { value: '1', label: 'Hug Layout' },
          { value: '2', label: 'Option 2' },
          { value: '3', label: 'Option 3' },
        ]}
      />
    </div>
    <div>
      <h4>Fill 布局</h4>
      <SingleSelect
        layout="fill"
        defaultValue="1"
        options={[
          { value: '1', label: 'Fill Layout' },
          { value: '2', label: 'Option 2' },
          { value: '3', label: 'Option 3' },
        ]}
      />
    </div>
  </div>
);

export default Demo;