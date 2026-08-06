import { CozAvatar, Badge } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex flex-row gap-2 p-4">
    <Badge count={33}>
      <CozAvatar color="orange" type="bot" size="lg">
        BD
      </CozAvatar>
    </Badge>
    <Badge count="beta">
      <CozAvatar color="green" type="bot" size="lg">
        BD
      </CozAvatar>
    </Badge>
    <Badge count={1000} overflowCount={999}>
      <CozAvatar color="orange" type="bot" size="lg">
        BD
      </CozAvatar>
    </Badge>
  </div>
);

export default Demo;