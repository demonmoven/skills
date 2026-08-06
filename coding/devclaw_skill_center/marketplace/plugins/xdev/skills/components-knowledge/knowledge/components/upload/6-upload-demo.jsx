import React from 'react';
import { Upload, Button } from '@coze-arch/coze-design';
import { IconCozUpload } from '@coze-arch/coze-design/icons';

const Demo = () => {
    let action = 'https://api.semi.design/upload';
    let imageOnly = 'image/*';
    let videoOnly = 'video/*';
    let fileLimit = '.pdf,.png,.jpeg';
    return (
        <>
            <Upload action={action} accept={imageOnly} style={{ marginBottom: 12 }}>
                <Button icon={<IconCozUpload />} theme="light">
                    上传图片
                </Button>
            </Upload>
            <Upload action={action} accept={videoOnly} style={{ marginBottom: 12 }}>
                <Button icon={<IconCozUpload />} theme="light">
                    上传视频
                </Button>
            </Upload>
            <Upload action={action} accept={fileLimit}>
                <Button icon={<IconCozUpload />} theme="light">
                    上传 PDF, PNG, JPEG
                </Button>
            </Upload>
        </>
    );
};

export default Demo;
