import React from 'react';
import { Upload, Image } from '@coze-arch/coze-design';
import { IconCozPlus } from '@coze-arch/coze-design/icons';

const Demo = () => {
    let action = 'https://api.semi.design/upload';
    const defaultFileList = [
        {
            uid: '1',
            name: 'image-1.jpg',
            status: 'success',
            size: '130KB',
            preview: true,
            url:
                'https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/abstract.jpg',
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
                picHeight={110}
                picWidth={200}
                renderThumbnail={(file) => (<Image src={file.url} width={200} height={110} />)}
            >
                <IconCozPlus size="extra-large" style={{ margin: 4 }} />
                点击添加图片
            </Upload>
        </>
    );
};

export default Demo;
