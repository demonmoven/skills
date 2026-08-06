import { Typography } from '@coze-arch/coze-design';

const { Title } = Typography;

const Demo = () => (
  <div className="flex flex-col gap-2">
    <Title fontSize="28px">COZTitle28（特大标题）</Title>
    <Title fontSize="20px">COZTitle20（大标题）</Title>
    <Title fontSize="16px">COZTitle16（标题）</Title>
    <Title fontSize="14px">COZTitle14（小标题）</Title>
    <Title fontSize="12px">COZTitle12（特小标题）</Title>
  </div>
);

export default Demo;