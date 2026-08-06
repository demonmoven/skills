import React, { useState } from 'react';
import { OverflowList, Tag, Slider } from '@coze-arch/coze-design';
import { IconCozClock, IconCozPin, IconCozImageflow, IconCozClock, IconCozEdit, IconCozFolder } from '@coze-arch/coze-design/icons';

const Demo = () => {
    const [width, setWidth] = useState(100);
    const renderOverflow = items => {
        return items.map(item => <Tag style={{ marginRight: 8, marginLeft: 8, flex: '0 0 auto', fontVariantNumeric: 'tabular-nums' }} key={item.key}>+{item.length}</Tag>);
    };
    const renderItem = (item, ind) => {
        return (
            <span key={item.key} className="item-cls">
                <Tag color="blue" style={{ marginRight: 8, flex: '0 0 auto' }}>
                    {item.icon}
                    {item.key}
                </Tag>
            </span>
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
                <OverflowList
                    items={items}
                    renderMode="scroll"
                    overflowRenderer={renderOverflow}
                    visibleItemRenderer={renderItem}
                />
            </div>
        </div>
    );
};

export default Demo;
