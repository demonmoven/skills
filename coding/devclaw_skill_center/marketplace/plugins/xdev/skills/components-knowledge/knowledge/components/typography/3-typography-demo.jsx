import { Typography } from '@coze-arch/coze-design';

const { Text } = Typography;

const Demo = () => (
  <div className="flex flex-col gap-2">
    <Text fontSize="16px">COZText16（大文字）</Text>
    <Text fontSize="14px">COZText14（标准文字）</Text>
    <Text fontSize="12px">COZText12（小文字）</Text>
    <Text fontSize="10px">COZText10（特小文字）</Text>
  </div>
);

export default Demo;