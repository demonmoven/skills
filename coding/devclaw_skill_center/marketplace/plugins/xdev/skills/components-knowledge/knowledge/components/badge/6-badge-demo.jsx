import { CozAvatar, Badge } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex flex-row gap-4 p-4">
    <Badge count={99}>
      <CozAvatar color="orange" type="bot" size="lg">
        BD
      </CozAvatar>
    </Badge>
    <Badge count={100}>
      <CozAvatar color="orange" type="bot" size="lg">
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