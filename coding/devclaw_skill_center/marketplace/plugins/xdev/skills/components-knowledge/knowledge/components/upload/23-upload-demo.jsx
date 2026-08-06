import React, { useState } from 'react';
import { Upload, Select, RadioGroup, Radio } from '@coze-arch/coze-design';
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
    const [hotSpotLocation, setLocation] = useState('end');
    return (
        <>
            <RadioGroup
                value={hotSpotLocation}
                type='button'
                onChange={e => setLocation(e.target.value)}>
                <Radio value='start'>start</Radio>
                <Radio value='end'>end</Radio>
            </RadioGroup>
            <hr />
            <Upload
                action={action}
                listType="picture"
                showPicInfo
                accept="image/*"
                multiple
                hotSpotLocation={hotSpotLocation}
                defaultFileList={defaultFileList}
                onPreviewClick={handlePreview}
            >
                <IconCozPlus size="extra-large" />
            </Upload>
        </>
    );
};

export default Demo;
