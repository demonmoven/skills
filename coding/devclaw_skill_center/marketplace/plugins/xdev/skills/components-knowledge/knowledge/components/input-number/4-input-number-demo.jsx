import { CozInputNumber } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div className="flex flex-col gap-4">
      <CozInputNumber prefix="¥" defaultValue={100} />
      <CozInputNumber suffix="%" defaultValue={50} min={0} max={100} />
      <CozInputNumber prefix="$" suffix="USD" defaultValue={99.99} />
    </div>
  );
};

export default Demo;