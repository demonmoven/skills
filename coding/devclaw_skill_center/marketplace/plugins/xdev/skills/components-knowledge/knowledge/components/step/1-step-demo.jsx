import { CozSteps } from '@coze-arch/coze-design';

const Demo = () => (
  <CozSteps current={1} type="basic">
    <CozSteps.Step title="已完成" description="这是一段描述文本" />
    <CozSteps.Step title="进行中" description="这是一段描述文本" />
    <CozSteps.Step title="待进行" description="这是一段描述文本" />
  </CozSteps>
);

export default Demo;