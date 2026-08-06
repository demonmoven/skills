import React from 'react';
import { Steps, Button } from '@coze-arch/coze-design';

class App extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            current: 0,
        };
    }

    next() {
        const current = this.state.current + 1;
        this.setState({ current });
    }

    prev() {
        const current = this.state.current - 1;
        this.setState({ current });
    }

    render() {
        const { current } = this.state;
        const { Step } = Steps;
        const steps = [
            {
                title: 'First',
                content: 'First-content',
            },
            {
                title: 'Second',
                content: 'Second-content',
            },
            {
                title: 'Last',
                content: 'Last-content',
            },
        ];

        return (
            <div>
                <Steps type="basic" current={current} onChange={(i)=>console.log(i)}>
                    {steps.map(item => (
                        <Step key={item.title} title={item.title} />
                    ))}
                </Steps>
                <div className="steps-content" style={{ marginTop: 4, marginBottom: 4 }}>{steps[current].content}</div>
                <div className="steps-action">
                    {current < steps.length - 1 && (
                        <Button type="primary" onClick={() => this.next()}>
                            Next
                        </Button>
                    )}
                    {current === steps.length - 1 && (
                        <Button type="primary" onClick={() => console.log('Processing complete!')}>
                            Done
                        </Button>
                    )}
                    {current > 0 && (
                        <Button style={{ marginLeft: 8 }} onClick={() => this.prev()}>
                            Previous
                        </Button>
                    )}
                </div>
            </div>
        );
    }
}

const Demo = () => <App />;

export default Demo;
