function domain(){
    let domain = '';
    if (process.env.NODE_ENV === 'development') {
        domain = 'http://localhost:6789/api';
    }
    if (process.env.NODE_ENV === 'production') {
        domain = 'https://go-unused-code-analyzer.byted.org/api';
    }
    return domain
}

export {domain}