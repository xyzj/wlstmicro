# wlst micro

基于ETCD和Gin Web Framework的五零盛同微服务框架。

具备mysql，mssql，rabbitmq，redis访问支持

## 清理所有提交的内容
git filter-branch --force --index-filter 'git rm --cached --ignore-unmatch *.exe* _apidoc.js' --prune-empty --tag-name-filter cat -- --all