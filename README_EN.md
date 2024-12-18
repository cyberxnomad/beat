## cron

[中文](README.md) | English

Based on project: https://github.com/robfig/cron (v3)

Functions:  

- Default time expression: [year] [month] [day] [weekday] [hour] [minute] [second]  
  - Customized time expressions can be supported via the layout parameter in the parser.

- Allowed symbols: `,`, `-`, `/`, `*`.  
  - Not supported `? `  

- Expressions do not support time zones currently.  

- Expressions like `@monthly`, `@weekly`, etc. are not supported.  

- DST (Daylight Saving Time) is not supported.  

TODO.  

- [x] custom logger support  
- [ ] Expressions support time zones  
- [ ] support for querying the current job  
