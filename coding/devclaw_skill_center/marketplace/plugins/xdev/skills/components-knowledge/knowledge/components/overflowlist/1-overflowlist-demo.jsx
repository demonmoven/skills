import React, { useState } from 'react';
import { OverflowList, Tag, Slider } from '@coze-arch/coze-design';
import { IconCozClock, IconCozPin, IconCozImageflow, IconCozClock, IconCozEdit, IconCozFolder } from '@coze-arch/coze-design/icons';

const Demo = () => {
    const [width, setWidth] = useState(100);
    const renderOverflow = items => {
        return items.length ? <Tag style={{ flex: '0 0 auto', fontVariantNumeric: 'tabular-nums' }}>+{items.length}</Tag> : null;
    };
    const renderItem = (item, ind) => {
        return (
            <Tag color="blue" key={item.key} style={{ marginRight: 8, flex: '0 0 auto' }}>
                {item.icon}
                {item.key}
            </Tag>
        );
    };

    const items = [
        { icon: <IconCozClock style={{ marginRight: 4 }} />, key: 'alarm' },
        { icon: <IconCozPin style={{ marginRight: 4 }} />, key: 'bookmark' },
        { icon: <IconCozImageflow style={{ marginRight: 4 }} />, key: 'camera' },
        { icon: <IconCozClock style={{ marginRight: 4 }} />, key: 'duration' },
        { icon: <IconCozEdit style={{ marginRight: 4 }} />, key: 'edit' },
        { icon: <IconCozFolder style={{ marginRight: 4 }} />, key: 'folder' },
    ];

    return (
        <div>
            <Slider step={1} value={width} onChange={value => setWidth(value)} />
            <br />
            <br />
            <div style={{ width: `${width}%` }}>
                <OverflowList items={items} overflowRenderer={renderOverflow} visibleItemRenderer={renderItem} />
            </div>
        </div>
    );
};

export default Demo;
