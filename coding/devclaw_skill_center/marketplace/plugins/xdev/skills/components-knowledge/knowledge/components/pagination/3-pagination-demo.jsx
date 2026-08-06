import { CozPagination } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex-col space-y-4">
    <CozPagination total={100} size="default" />
    <CozPagination total={100} size="small" />
  </div>
);

export default Demo;