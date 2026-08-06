import React from 'react';
import { Avatar, Calendar } from '@coze-arch/coze-design';

const Demo = () => {
    const displayValue = new Date(2023, 4, 14);

    const renderDateDisplay = date => {
        const colors = ["amber", "blue", "cyan", "green", "grey", "indigo", "lime"];
        return <div><Avatar color={colors[date.getDay()]} size="small">{date.getDate()}</Avatar></div>;
    };

    return <Calendar height={400} mode="week" displayValue={displayValue} renderDateDisplay={renderDateDisplay} />;
};

export default Demo;
