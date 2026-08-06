import React from 'react';
import { Upload, Button } from '@coze-arch/coze-design';
import { IconCozUpload } from '@coze-arch/coze-design/icons';

const Demo = () => {
    let action = 'https://api.semi.design/upload';
    return (
        <>
            <Upload action={action} directory>
                <Button icon={<IconCozUpload />} theme="light">
                    上传文件夹
                </Button>
            </Upload>
        </>
    );
};

export default Demo;
