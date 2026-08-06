import { InputCode } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div className="flex flex-col gap-4">
      <InputCode type="text" placeholder="文本模式" />
      <InputCode type="password" placeholder="密码模式" />
    </div>
  );
};

export default Demo;