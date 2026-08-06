import React from 'react';
import { Highlight } from '@coze-arch/coze-design';

const Demo = () => {
    return (
        <h2>
            <Highlight
                component='span'
                sourceString='从 Semi Design 到 Any Design  快速定义你的设计系统，并应用在设计稿和代码中'
                searchWords={[
                    { text: 'Semi', style: { backgroundColor: 'rgba(var(--semi-teal-5), 1)', color: 'rgba(var(--semi-white), 1)', padding: 4 }, className: 'keyword1' },
                    { text: '设计系统', style: { backgroundColor: 'var(--semi-color-primary)', color: 'rgba(var(--semi-white), 1)', padding: 4 }, className: 'keyword2' },
                    { text: '设计稿和代码', style: { backgroundColor: 'rgba(var(--semi-violet-5), 1)', color: 'rgba(var(--semi-white), 1)', padding: 4 }, className: 'keyword3' },
                ]}
                highlightStyle={{ borderRadius: 4 }}
            />
        </h2>
    );
};

export default Demo;
