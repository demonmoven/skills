import React from 'react';
import { Anchor } from '@coze-arch/coze-design';

const Demo = () => (
    <Anchor size={'default'}>
        <Anchor.Link href="#组件" title="组件" />
        <Anchor.Link href="#设计语言" title="设计语言" />
        <Anchor.Link href="#物料平台" title="物料平台" />
        <Anchor.Link href="#主题商店" title="主题商店" />
    </Anchor>
);

export default Demo;
