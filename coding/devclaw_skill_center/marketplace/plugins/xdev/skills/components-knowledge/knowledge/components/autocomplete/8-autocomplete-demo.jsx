import { useState } from 'react';
import { AutoComplete, Empty } from '@coze-arch/coze-design';
import { IllustrationNoContent } from '@coze-arch/coze-design/illustrations';

const Demo = () => {
    let [data, setData] = useState([]);
    const [loading, setLoading] = useState(false);

    const fetchData = v => {
        setLoading(true);
        setTimeout(() => {
            if (!v) {
                setData([]);
                setLoading(false);
                return;
            }
            setData(() => {
                const res = Array.from(Array(5)).map(c => Math.random());
                return res;
            });
            setLoading(false);
        }, 1000);
    };

    return (
        <AutoComplete
            loading={loading}
            data={data}
            emptyContent={<Empty style={{ padding: 12, width: 300 }} image={<IllustrationNoContent style={{ width: 150, height: 150 }}/>} description={'暂无内容'} />}
            onSearch={fetchData}
        />
    );
};

export default Demo;
