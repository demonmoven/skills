import { CozAvatar } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
      <CozAvatar.AvatarGroup size="small">
        <CozAvatar>A</CozAvatar>
        <CozAvatar>B</CozAvatar>
        <CozAvatar>C</CozAvatar>
      </CozAvatar.AvatarGroup>

      <CozAvatar.AvatarGroup>
        <CozAvatar>A</CozAvatar>
        <CozAvatar>B</CozAvatar>
        <CozAvatar>C</CozAvatar>
      </CozAvatar.AvatarGroup>

      <CozAvatar.AvatarGroup size="medium">
        <CozAvatar>A</CozAvatar>
        <CozAvatar>B</CozAvatar>
        <CozAvatar>C</CozAvatar>
      </CozAvatar.AvatarGroup>

      <CozAvatar.AvatarGroup size="large">
        <CozAvatar>A</CozAvatar>
        <CozAvatar>B</CozAvatar>
        <CozAvatar>C</CozAvatar>
      </CozAvatar.AvatarGroup>
    </div>
  );
};

export default Demo;