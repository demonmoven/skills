import { Search } from '@coze-arch/coze-design';

const Demo = () => (
  <Search
    placeholder="输入完成后触发搜索"
    onSearch={value => console.log('搜索值:', value)}
    onChange={(value, e) => console.log('输入值:', value)}
  />
);

export default Demo;