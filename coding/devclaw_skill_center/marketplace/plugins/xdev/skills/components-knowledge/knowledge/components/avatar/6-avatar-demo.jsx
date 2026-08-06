import { CozAvatar } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div style={{ display: 'flex', gap: 12 }}>
      <CozAvatar
        onClick={() => console.log('Avatar clicked')}
        onMouseEnter={() => console.log('Mouse entered')}
        onMouseLeave={() => console.log('Mouse left')}
        hoverMask
      >
        E
      </CozAvatar>
      <CozAvatar
        src="https://sf6-cdn-tos.douyinstatic.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/avatarDemo.jpeg"
        onClick={() => console.log('Image avatar clicked')}
        hoverMask
      />
    </div>
  );
};

export default Demo;