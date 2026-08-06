import { Search } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
    <Search placeholder="请输入搜索内容" />
    <Search placeholder="隐藏搜索图标" hideIcon />
    <Search placeholder="禁用状态" disabled />
  </div>
);

export default Demo;