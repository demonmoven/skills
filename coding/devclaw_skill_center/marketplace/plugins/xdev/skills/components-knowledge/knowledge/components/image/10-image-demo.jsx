import React, { useMemo } from 'react';
import { Image, ImagePreview } from '@coze-arch/coze-design';

const Demo = () => {
    const srcList = useMemo(() => ([
        "https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/abstract.jpg",
        "https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/sky.jpg",
    ]), []);

    return (
        <>  
            <ImagePreview
                renderHeader={(title) => (
                    <div 
                        style={{ width: "100%", height: "100%", display: "flex", alignItems: "center", justifyContent: "center" }}>
                        <span style={{ background: "black", padding: '0 10px' }}>自定义标题:{title}</span>
                    </div>
                )}
            >
                <div >
                    {srcList.map((src, index) => {
                        return (
                            <Image 
                                key={index} 
                                src={src} 
                                width={200} 
                                alt={'lamp' + (index + 1)} 
                                preview={{
                                    previewTitle: 'lamp' + (index + 1),
                                }}
                                style={{ marginRight: 5 }} 
                            />
                        );
                    })}
                </div>
            </ImagePreview>
        </>
    );
};

export default Demo;
