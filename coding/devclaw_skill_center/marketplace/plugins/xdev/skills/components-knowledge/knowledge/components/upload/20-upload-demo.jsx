import React from 'react';
import { Upload, Image } from '@coze-arch/coze-design';
import { IconCozPlus } from '@coze-arch/coze-design/icons';

const Demo = () => {
    let action = 'https://api.semi.design/upload';
    const defaultFileList = [
        {
            uid: '1',
            name: 'music.png',
            status: 'success',
            size: '130KB',
            preview: true,
            url:
                'https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/Resso.png',
        }
    ];
    return (
        <>
            <Upload
                action={action}
                listType="picture"
                accept="image/*"
                multiple
                defaultFileList={defaultFileList}
                renderThumbnail={(file) => (<Image src={file.url} />)}
            >
                <IconCozPlus size="extra-large" />
            </Upload>
        </>
    );
};

export default Demo;
