import React from 'react';
import { Anchor } from '@coze-arch/coze-design';

const Demo = () => {
    const getContainer = () => {
        return document.querySelector('window');
    };
    return (
        <div>
            <Anchor
                showTooltip={true}
                getContainer={getContainer}
                targetOffset={60}
                offsetTop={100}
            >
                <Anchor.Link href="#显示工具提示" title="工具提示是一个有用的工具，它可以在文字缩略时展示全部内容。" />
                <Anchor.Link href="#组件" title="组件" />
                <Anchor.Link href="#设计语言" title="设计语言" />
                <Anchor.Link href="#物料平台" title="物料平台" />
                <Anchor.Link href="#主题商店" title="主题商店" />
            </Anchor>
        </div>
    );
};

export default Demo;
