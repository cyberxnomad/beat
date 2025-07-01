## Beat

中文 | [English](README_EN.md)  

> [!IMPORTANT]  
> 为了防止和标准实现的 Cron 库混淆，我们将名称改为 Beat 。

Beat 是一个类似 cron 的定时库。

此程序并非标准的 cron 实现，仅兼容部分标准 cron 特性。  

基于项目：https://github.com/robfig/cron (v3)  

### 功能：  

- 支持秒
  - 默认时间表达式：`[month] [day] [weekday] [hour] [minute] [second]`  
  - 可通过 parser 中的 layout 参数来支持自定义时间表达式  

- 允许的符号：`,`(多个时间), `-`(范围), `/`(步长), `*`(通配)  
  - 不支持 `?`  

- 表达式中的月份仅支持数字，不支持形如 `Jan`、`Feb` 等形式；星期仅支持数字，不支持形如 `Mon`、`Tue` 等形式  

- ~~表达式暂不支持时区~~  

- 不支持形如 `@monthly`、`@weekly` 等表达式  

- 不支持 DST (夏令时)  

### TODO:  

- [x] 支持自定义 logger  
- [x] 表达式支持时区  
      例: `TZ=Asia/Shanghai * * * * * *`
- [ ] 支持查询当前任务  
