import React, { useState } from 'react';
import { PageError, PageLoading, PageNoAuth, PageNotFound, FullPage } from '@cozeloop/components';

const Demo = () => {
  const [status, setStatus] = useState('loading');

  return (
    <div>
      <div style={{ display: 'flex', gap: 8, marginBottom: 16 }}>
        <button onClick={() => setStatus('loading')}>Loading</button>
        <button onClick={() => setStatus('error')}>Error</button>
        <button onClick={() => setStatus('notfound')}>Not Found</button>
        <button onClick={() => setStatus('noauth')}>No Auth</button>
      </div>
      <div style={{ height: 300, position: 'relative' }}>
        {status === 'loading' ? <PageLoading tip="加载中..." /> : null}
        {status === 'error' ? <PageError /> : null}
        {status === 'notfound' ? <PageNotFound /> : null}
        {status === 'noauth' ? <PageNoAuth /> : null}
      </div>
    </div>
  );
};

export default Demo;
