import React from 'react';
import { Anchor } from '@coze-arch/coze-design';

const Demo = () => {
    const getContainer = () => {
        return document.querySelector('window');
    };
    return (
        <div>
            <Anchor
                railTheme={'primary'}
                getContainer={getContainer}
                targetOffset={60}
                offsetTop={100}
            >
                <Anchor.Link href="#尺寸" title="尺寸" />
                <Anchor.Link href="#滑轨主题" title="滑轨主题" />
                <Anchor.Link href="#设计语言" title="设计语言" />
                <Anchor.Link href="#物料平台" title="物料平台" />
                <Anchor.Link href="#主题商店" title="主题商店" />
            </Anchor>
        </div>
    );
};

export default Demo;
