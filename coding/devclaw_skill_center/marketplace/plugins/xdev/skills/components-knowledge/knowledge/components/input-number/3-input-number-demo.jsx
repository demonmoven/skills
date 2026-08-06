import { CozInputNumber } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div className="flex flex-col gap-4">
      <CozInputNumber
        defaultValue={1}
        step={0.1}
        shiftStep={1}
        min={0}
        max={10}
      />
      <CozInputNumber
        defaultValue={100}
        step={10}
        shiftStep={100}
        min={0}
        max={1000}
      />
    </div>
  );
};

export default Demo;