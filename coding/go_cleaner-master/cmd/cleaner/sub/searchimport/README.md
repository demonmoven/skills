# 子命令-查询某个module在这个仓库的所有引用

```shell
cd ${ProjectRootDirOfDemoA}

cleaner search --target=code.byted.org/demo/x --out=/tmp/import.txt # 搜索code.byted.org/demo/x在这个demo/a仓库中的所有引用，将结果存在import.txt中

# 该命令配合cleaner使用
cd ${ProjectRootDirOfDemoX}
cleaner --extra_import=/tmp/import.txt # 清理demo/x项目的代码，import.txt中的方法、类型等都会保留, 即demo/x在demo/a中的所有引用都会保留
```
