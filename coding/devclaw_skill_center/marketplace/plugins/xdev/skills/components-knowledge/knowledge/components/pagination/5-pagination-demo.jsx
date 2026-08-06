import { CozPagination } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex-col space-y-4">
    <CozPagination total={100} showQuickJumper />
    <CozPagination total={100} showQuickJumper showTotal />
  </div>
);

export default Demo;