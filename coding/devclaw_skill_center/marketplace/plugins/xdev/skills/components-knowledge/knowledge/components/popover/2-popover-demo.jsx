import { Popover, Button } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
    {[
      'top',
      'topLeft',
      'topRight',
      'left',
      'leftTop',
      'leftBottom',
      'right',
      'rightTop',
      'rightBottom',
      'bottom',
      'bottomLeft',
      'bottomRight',
    ].map(position => (
      <Popover
        key={position}
        content={`Position: ${position}`}
        position={position}
        showArrow
      >
        <Button>{position}</Button>
      </Popover>
    ))}
  </div>
);

export default Demo;