import { CozInputNumber } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div className="flex flex-col gap-4">
      <CozInputNumber size="small" defaultValue={1} />
      <CozInputNumber size="default" defaultValue={1} />
    </div>
  );
};

export default Demo;