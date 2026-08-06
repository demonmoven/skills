import React from 'react';
import { Upload, Button, Toast } from '@coze-arch/coze-design';
import { IconCozUpload } from '@coze-arch/coze-design/icons';

const Demo = () => {
    let action = 'https://api.semi.design/upload';

    return (
        <>
            <Upload
                action={action}
                maxSize={1024}
                minSize={200}
                onSizeError={(file, fileList) => Toast.error(`${file.name} size invalid`)}
            >
                <Button icon={<IconCozUpload />} theme="light">
                    点击上传（最小 200KB，最大 1MB）
                </Button>
            </Upload>
        </>
    );
};

export default Demo;
