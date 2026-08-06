import { CozPagination } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex-col space-y-4">
    <CozPagination total={100} layout="default" />
    <CozPagination total={100} layout="simple" />
  </div>
);

export default Demo;