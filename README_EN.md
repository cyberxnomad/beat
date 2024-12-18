## cron

[中文](README.md) | English  

> [!NOTE]  
> This program is not a standard cron implementation and is only compatible with part of the standard cron features.  

Based on: https://github.com/robfig/cron (v3)  

### Features:  

- Support year and second
  - Default time expression: `[year] [month] [day] [weekday] [hour] [minute] [second]`  
  - Customized time expressions can be supported via the layout parameter in the parser.  

- Allowed symbols: `,`, `-`, `/`, `*`.  
  - Not supported `? `  
  
- Months in expressions are numeric only, not in the form `Jan`, `Feb`, etc. Weeks are numeric only, not in the form `Mon`, `Tue`, etc.  

- Expressions do not support time zones currently.  

- Expressions like `@monthly`, `@weekly`, etc. are not supported.  

- DST (Daylight Saving Time) is not supported.  

### TODO:  

- [x] custom logger support  
- [ ] Expressions support time zones  
- [ ] support for querying the current job  
