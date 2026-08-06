import React from 'react';
import { Rating } from '@coze-arch/coze-design';
import { IconCozThumbsupFill } from '@coze-arch/coze-design/icons';

const Demo = () => (
    <div>
        <Rating style={{ color: 'red' }} character={(<IconCozThumbsupFill size="extra-large" />)} defaultValue={3}/>
        <br/>
        <br/>
        <Rating style={{ color: 'red' }} size={48} allowHalf character={(<IconCozThumbsupFill style={{ fontSize: 48 }} />)} defaultValue={3}/>
        <br/>
        <br/>
        <Rating character={'赞'} size={18} defaultValue={3}/>
        <br/>
        <br/>
        <Rating count={10} defaultValue={6}/>
    </div>
);

export default Demo;
