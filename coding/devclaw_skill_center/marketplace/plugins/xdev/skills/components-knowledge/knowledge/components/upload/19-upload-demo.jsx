import React from 'react';
import { Upload } from '@coze-arch/coze-design';
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
        },
        {
            uid: '2',
            name: 'brand.png',
            status: 'success',
            size: '130KB',
            preview: true,
            url:
                'https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/Resso.png',
        },
    ];
    return (
        <>
            <Upload action={action} listType="picture" showPicInfo accept="image/*" multiple defaultFileList={defaultFileList}>
                <IconCozPlus size="extra-large" />
            </Upload>
        </>
    );
};

export default Demo;
