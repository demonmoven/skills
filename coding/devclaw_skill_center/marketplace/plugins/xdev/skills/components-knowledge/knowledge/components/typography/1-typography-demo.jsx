import { Typography } from '@coze-arch/coze-design';

const { Title, Text, Paragraph, Numeral } = Typography;

const Demo = () => (
  <div className="flex flex-col gap-8">
    <Title heading={3}>标题Title</Title>
    <Text size="normal">文字Text</Text>
    <Paragraph size="normal">段落Paragraph</Paragraph>
    <Numeral className="coz-fg-primary" precision={2}>
      <p>数字Numeral：1.6111e1 K</p>
    </Numeral>
  </div>
);

export default Demo;