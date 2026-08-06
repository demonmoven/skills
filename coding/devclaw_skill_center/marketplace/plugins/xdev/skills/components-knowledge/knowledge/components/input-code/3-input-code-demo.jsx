import { InputCode } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div className="flex flex-col gap-4">
      <InputCode length={4} placeholder="4位验证码" />
      <InputCode length={6} placeholder="6位验证码" />
    </div>
  );
};

export default Demo;