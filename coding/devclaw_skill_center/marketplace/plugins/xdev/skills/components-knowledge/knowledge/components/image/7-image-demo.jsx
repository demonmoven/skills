import React, { useMemo } from 'react';
import { Image, ImagePreview } from '@coze-arch/coze-design';

const Demo = () => {
    const srcList = useMemo(() => ([
        "https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/abstract.jpg",
        "https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/sky.jpg",
        "https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/greenleaf.jpg",
    ]), []);

    return ( 
        <>
            <div 
                id="container" 
                style={{ 
                    height: 400, 
                    position: "relative" 
                }} 
            >
                <ImagePreview
                    getPopupContainer={() => {
                        const node = document.getElementById("container");
                        return node;
                    }}
                    style={{
                        height: '100%',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
        
                    }}
                >
                    {srcList.map((src, index) => {
                        return (
                            <Image 
                                key={index} 
                                src={src} 
                                width={200} 
                                alt={'lamp' + (index + 1)}
                                style={{ marginRight: 5 }} 
                            />
                        );
                    })}
                </ImagePreview>
            </div>
        </>
    );
};

export default Demo;
