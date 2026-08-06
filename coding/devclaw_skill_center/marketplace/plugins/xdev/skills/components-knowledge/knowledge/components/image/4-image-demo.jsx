import React from 'react';
import { Image } from '@coze-arch/coze-design';

const Demo = () => {
     return ( 
         <Image
             width={300}
             height={200}
             src={'https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/abstract-small.jpeg'}
             preview={{
                 src: 'https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/abstract-big.png'
             }}
         />
     );
};

export default Demo;
