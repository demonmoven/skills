---
description: 制作每周AI coding分析报告
argument-hint: file_path
---

读取file_path,做如下分析

## Step1 分析openspec旅程
分析`命中的OpenSpec_Prompts`这一列，总结提炼用户在开发这个需求的过程中，使用openspec ai coding的心路历程和主要问题，写入新的一列“openspec旅程分析”，删掉原命中的OpenSpec_Prompts

## Step2 计算额外指标
- 需求AI开发使用率：openspec+vide coding的需求数/ 总需求数

## Step3 输出飞书文档
把需求AI开发使用率和表格输出到一个飞书文档中