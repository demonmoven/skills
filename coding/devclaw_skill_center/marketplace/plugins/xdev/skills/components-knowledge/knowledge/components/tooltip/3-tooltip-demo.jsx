import { Tooltip, Button } from '@coze-arch/coze-design';

const Demo = () => {
  const positions = [
    ['topLeft', 'TL'],
    ['top', 'Top'],
    ['topRight', 'TR'],
    ['leftTop', 'LT'],
    ['left', 'Left'],
    ['leftBottom', 'LB'],
    ['rightTop', 'RT'],
    ['right', 'Right'],
    ['rightBottom', 'RB'],
    ['bottomLeft', 'BL'],
    ['bottom', 'Bottom'],
    ['bottomRight', 'BR'],
  ];

  return (
    <div className="grid grid-cols-3 gap-4">
      {positions.map(([pos, text]) => (
        <Tooltip key={pos} content={`Position: ${pos}`} position={pos}>
          <Button className="w-full">{text}</Button>
        </Tooltip>
      ))}
    </div>
  );
};

export default Demo;