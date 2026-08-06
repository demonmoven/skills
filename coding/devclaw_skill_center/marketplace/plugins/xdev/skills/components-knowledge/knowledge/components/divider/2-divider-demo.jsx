import React from 'react';
import { Divider, Typography } from '@coze-arch/coze-design';
import { IconCozAi } from '@coze-arch/coze-design/icons';

const Demo = () => {

    return (
        <div>
            <Divider margin='12px' align='left'>
                这是居左文字
            </Divider>

            <Divider margin='12px' align='center'>
                这是居中文字
            </Divider>

            <Divider margin='12px' align='right'>
                这是居右文字
            </Divider>

            <Divider margin='12px'>
                <IconCozAi />
            </Divider>
        </div>
    );
};


export default Demo;
