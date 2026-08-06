import { Search } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
    <Search width={200} placeholder="固定宽度 200px" />
    <Search width="50%" placeholder="相对宽度 50%" />
  </div>
);

export default Demo;