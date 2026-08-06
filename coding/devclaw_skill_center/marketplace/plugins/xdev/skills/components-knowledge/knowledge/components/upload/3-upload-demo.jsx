import React from 'react';
import { Upload } from '@coze-arch/coze-design';
import { IconCozPlus } from '@coze-arch/coze-design/icons';

const Demo = () => {
    const action = 'https://api.semi.design/upload';
    const getPrompt = (pos, isListType) => {
        let basicStyle = { display: 'flex', alignItems: 'center', color: 'grey', height: isListType ? '100%' : 32 };
        let marginStyle = {
            left: { marginRight: 10 },
            right: { marginLeft: 10 },
        };
        let style = { ...basicStyle, ...marginStyle[pos] };

        return <div style={style}>请上传认证材料</div>;
    };
    const defaultFileList = [
        {
            uid: '1',
            name: 'dy.jpeg',
            status: 'success',
            size: '130kb',
            url:
                'https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/dy.png',
        },
        {
            uid: '5',
            name: 'resso.jpeg',
            percent: 50,
            size: '222kb',
            url:
                'https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/Resso.png',
        },
    ];
    const positions = ['right', 'bottom'];
    return (
        <>
            {positions.map((pos, index) => (
                <>
                    {index ? (
                        <div
                            style={{ marginBottom: 12, marginTop: 12, borderBottom: '1px solid var(--semi-color-border)' }}
                        ></div>
                    ) : null}
                    <Upload
                        action={action}
                        prompt={getPrompt(pos, true)}
                        promptPosition={pos}
                        listType="picture"
                        defaultFileList={defaultFileList}
                    >
                        <IconCozPlus size="extra-large" />
                    </Upload>
                </>
            ))}
        </>
    );
};

export default Demo;
