import { CozPagination } from '@coze-arch/coze-design';

const Demo = () => (
  <CozPagination
    total={100}
    currentPage={1}
    pageSize={10}
    onChange={(page, pageSize) => console.log(page, pageSize)}
  />
);

export default Demo;