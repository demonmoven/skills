import React, { useState } from 'react';
import { Collapsible, Button } from '@coze-arch/coze-design';

const Demo = () => {
    const [isOpen, setOpen] = useState(false);
    const maskStyle = isOpen
        ? {}
        : {
            WebkitMaskImage:
                    'linear-gradient(to bottom, black 0%, rgba(0, 0, 0, 1) 60%, rgba(0, 0, 0, 0.2) 80%, transparent 100%)',
        };
    const collapsed = (
        <ul>
            <li>
                <p>Semi Design 以内容优先进行设计。</p>
            </li>
            <li>
                <p>更容易地自定义主题。</p>
            </li>
            <li>
                <p>适用国际化场景。</p>
            </li>
            <li>
                <p>效率场景加入人性化关怀。</p>
            </li>
        </ul>
    );
    const toggle = () => {
        setOpen(!isOpen);
    };
    const linkStyle = {
        position: 'absolute',
        left: 0,
        right: 0,
        textAlign: 'center',
        bottom: -10,
        fontWeight: 700,
        cursor: 'pointer',
    };
    return (
        <>
            <Button onClick={toggle}>Toggle</Button>
            <div style={{ position: 'relative' }}>
                <Collapsible isOpen={isOpen} collapseHeight={60} style={{ ...maskStyle }}>
                    {collapsed}
                </Collapsible>
                {isOpen ? null : (
                    <a onClick={toggle} style={{ ...linkStyle }}>
                        + Show More
                    </a>
                )}
            </div>
        </>
    );
};

export default Demo;
