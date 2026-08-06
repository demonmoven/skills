# 子命令-删除废弃接口



```shell
cd ${ProjectRootDir}

cleaner endpoint -d --ep=createdraft,modifydraft # -d 则删除整个接口，否则会留一个空实现 return nil, nil
```

