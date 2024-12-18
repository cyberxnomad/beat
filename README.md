## cron

中文 | [English](README_EN.md)  

> [!NOTE]  
> 此程序并非标准的 cron 实现，仅兼容部分标准 cron 特性。  

基于项目：https://github.com/robfig/cron (v3)  

### 功能：  

- 支持年、秒
  - 默认时间表达式：`[year] [month] [day] [weekday] [hour] [minute] [second]`  
  - 可通过 parser 中的 layout 参数来支持自定义时间表达式  

- 允许的符号：`,`(多个时间), `-`(范围), `/`(步长), `*`(通配)  
  - 不支持 `?`  

- 表达式中的月份仅支持数字，不支持形如 `Jan`、`Feb` 等形式；星期仅支持数字，不支持形如 `Mon`、`Tue` 等形式  

- 表达式暂不支持时区  

- 不支持形如 `@monthly`、`@weekly` 等表达式  

- 不支持 DST (夏令时)  

### TODO:  

- [x] 支持自定义 logger  
- [ ] 表达式支持时区  
- [ ] 支持查询当前任务  
