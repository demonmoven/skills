import { CozPagination } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex-col space-y-4">
    <CozPagination
      total={100}
      showTotal
      showSizeChanger
      totalText={(total, range) => `${range[0]}-${range[1]} of ${total} items`}
    />
  </div>
);

export default Demo;