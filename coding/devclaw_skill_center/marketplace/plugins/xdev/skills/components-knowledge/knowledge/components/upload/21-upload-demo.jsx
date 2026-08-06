import React from 'react';
import { Upload } from '@coze-arch/coze-design';
import { IconCozPlus, IconCozEye } from '@coze-arch/coze-design/icons';

const Demo = () => {
    let action = 'https://api.semi.design/upload';
    const defaultFileList = [
        {
            uid: '1',
            name: 'resso.png',
            status: 'success',
            size: '130KB',
            preview: true,
            url:
                'https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/Resso.png',
        },
    ];
    const handlePreview = (file) => {
        const feature = "width=300,height=300";
        window.open(file.url, 'imagePreview', feature);
    };
    return (
        <>
            <Upload
                action={action}
                listType="picture"
                showPicInfo
                accept="image/*"
                multiple
                defaultFileList={defaultFileList}
                onPreviewClick={handlePreview}
                renderPicPreviewIcon={()=><IconCozEye style={{ color: 'var(--semi-color-white)', fontSize: 24 }} />}
            >
                <IconCozPlus size="extra-large" />
            </Upload>
        </>
    );
};

export default Demo;
