import { CozInputNumber } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div className="flex flex-col gap-4">
      {/* 带前后缀的滑块控制 */}
      <CozInputNumber
        prefix="¥"
        defaultValue={10}
        suffix="元"
        sliderControl={true}
        step={0.01}
      />

      {/* 滑块控制与按钮组合 */}
      <CozInputNumber
        prefix="$"
        suffix="USD"
        step={0.01}
        sliderControl={true}
        hideButtons={false}
      />

      {/* 外置按钮的滑块控制 */}
      <CozInputNumber
        prefix="€"
        suffix="EUR"
        step={0.01}
        sliderControl={true}
        hideButtons={false}
        innerButtons={false}
      />
    </div>
  );
};

export default Demo;