import React from 'react';
import { Upload, Button, Image } from '@coze-arch/coze-design';
import { IconCozUpload, IconCozDocument } from '@coze-arch/coze-design/icons';

const Demo = () => {
    let action = 'https://api.semi.design/upload';
    const defaultFileList = [
        {
            uid: '1',
            name: 'dyBag.png',
            status: 'success',
            size: '130KB',
            url: 'https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/edit-bag.jpeg',
        },
        {
            uid: '2',
            name: 'dyBag2.png',
            status: 'success',
            size: '130KB',
            url: 'https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/edit-bag.jpeg',
        },
    ];
    return (
        <Upload
            defaultFileList={defaultFileList}
            action={action}
            previewFile={file => file.uid === '1' ? <IconCozDocument size="large" /> : <Image src={file.url} />}
        >
            <Button icon={<IconCozUpload />} theme="light">
                点击上传
            </Button>
        </Upload>
    );
};

export default Demo;
