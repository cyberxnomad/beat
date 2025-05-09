package beat

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type LayoutField uint32

const (
	Year LayoutField = 1 << iota
	Month
	Dom
	Dow
	Hour
	Minute
	Second
)

var DefaultLayout = []LayoutField{Year, Month, Dom, Dow, Hour, Minute, Second}

type Parser struct {
	layout         []LayoutField
	defaultLoction *time.Location // 缺省时区，解析时未指定时区则以该参数时区解析
}

type SchedTime struct {
	Year   [2]uint64 // 年
	Month  uint64    // 月
	Dom    uint64    // 日
	Dow    uint64    // 星期，0=星期日
	Hour   uint64    // 时
	Minute uint64    // 分
	Second uint64    // 秒

	location *time.Location
}

var defaultParser = NewParser()

func NewParser(opts ...ParserOption) *Parser {
	p := new(Parser)
	p.layout = DefaultLayout
	p.defaultLoction = time.Local

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// 获取域的限制范围
func (f LayoutField) bounds() (min, max int) {
	switch f {
	case Year:
		min = 1970
		max = min + 127

	case Month:
		min = 1
		max = 12

	case Dom:
		min = 1
		max = 31

	case Dow:
		min = 0
		max = 6

	case Hour:
		min = 0
		max = 23

	case Minute:
		min = 0
		max = 59

	case Second:
		min = 0
		max = 59
	}

	return
}

// 解析时间表达式
func (p *Parser) Parse(exp string) (Schedule, error) {
	fields := strings.Fields(exp)

	if len(fields) < len(p.layout) {
		return nil, fmt.Errorf("%w: invalid number of fields", ErrInvalidExp)
	}

	st := new(SchedTime)
	st.location = p.defaultLoction
	offset := 0

	if loc, found := strings.CutPrefix(fields[0], "TZ="); found {
		if len(fields)-1 < len(p.layout) {
			return nil, fmt.Errorf("%w: invalid number of fields", ErrInvalidExp)
		}

		location, err := time.LoadLocation(loc)
		if err != nil {
			return nil, fmt.Errorf("bad location '%s': %v", loc, err)
		}

		st.location = location
		offset = 1
	}

	for i := range p.layout {
		bits, err := parseField(fields[i+offset], p.layout[i])
		if err != nil {
			return nil, err
		}

		switch p.layout[i] {
		case Year:
			st.Year = bits

		case Month:
			st.Month = bits[0]

		case Dom:
			st.Dom = bits[0]

		case Dow:
			st.Dow = bits[0]

		case Hour:
			st.Hour = bits[0]

		case Minute:
			st.Minute = bits[0]

		case Second:
			st.Second = bits[0]
		}
	}

	return st, nil
}

// 解析域
//
// 支持符号：, - * /
func parseField(field string, lf LayoutField) ([2]uint64, error) {
	ranges := strings.Split(field, ",")
	min, max := lf.bounds()

	bits := [2]uint64{}
	err := error(nil)
	for _, exp := range ranges {
		start, end, step := 0, 0, 0
		// 分离范围和步进
		rangeAndStep := strings.Split(exp, "/")
		// 分离范围起始和结束
		lowAndHigh := strings.Split(rangeAndStep[0], "-")

		if lowAndHigh[0] == "*" {
			if len(lowAndHigh) != 1 {
				// 不允许出现类似 *-2 的表达式
				return [2]uint64{}, fmt.Errorf("%w: %s", ErrInvalidExp, exp)
			}
			// 若为通配符，则起始和结束分别为最小值和最大值
			start = min
			end = max
		} else {
			// 首个字符不是通配符，说明表达式中至少标明了起始值，尝试转换为整型
			start, err = strconv.Atoi(lowAndHigh[0])
			if err != nil {
				return [2]uint64{}, fmt.Errorf("%w: %s", ErrInvalidExp, err)
			}

			switch len(lowAndHigh) {
			case 1: // 长度为1，说明表达式中没有标明结束值
				end = start

			case 2: // 长度为2，说明表达式中标明了结束值
				end, err = strconv.Atoi(lowAndHigh[1])
				if err != nil {
					return [2]uint64{}, fmt.Errorf("%w: %s", ErrInvalidExp, err)
				}

			default: // 语法错误
				return [2]uint64{}, fmt.Errorf("%w: too many hyphens: %s", ErrInvalidExp, exp)
			}
		}

		switch len(rangeAndStep) {
		case 1: // 长度为1，则说明表达式中没有标明步长，默认步长为1
			step = 1

		case 2: // 长度为2，则说明表达式中含有步长
			step, err = strconv.Atoi(rangeAndStep[1])
			if err != nil {
				return [2]uint64{}, fmt.Errorf("%w: %s", ErrInvalidExp, err)
			}
			if step <= 0 {
				return [2]uint64{}, fmt.Errorf("%w: negative or zero step is not allowed", ErrInvalidExp)
			}

			// 表达式中没有标明结束值，则将结束值设为最大值
			if len(lowAndHigh) == 1 {
				end = max
			}
		default:
			return [2]uint64{}, fmt.Errorf("%w: too many slashes: %s", ErrInvalidExp, exp)
		}

		// 判断参数是否超出范围
		if start < min || end > max || start > end {
			return [2]uint64{}, fmt.Errorf("%w: out of range: %s", ErrInvalidExp, exp)
		}

		// 如果当前解析的域为年，则减去1970（以1970为起始年份）
		if lf == Year {
			start -= 1970
			end -= 1970
		}

		// 为有效位置1
		for i := start; i <= end; i += step {
			if i < 64 {
				bits[0] |= 1 << i
			} else {
				bits[1] |= 1 << (i - 64)
			}
		}
	}

	return bits, nil
}

// 获取下一个有效时间
func (st *SchedTime) Next(t time.Time) time.Time {
	// 检查时间域是否匹配，如果匹配，则进行下一个域的匹配。
	// 如果域不匹配，则增加该域的值。

	// 如果指定了时区，则将给定时间转换为 SchedTime 的时区。
	// 保存原始时区，以便找到时间后再转换回来。
	// 请注意，未指定时区的 SchedTime 将被视为本地时区。
	origLocation := t.Location()
	loc := st.location
	if loc == time.Local {
		loc = t.Location()
	}
	if st.location != time.Local {
		t = t.In(st.location)
	}

	// 匹配机制未匹配到时，将一直增加时间进行匹配，
	// 此值用于限制匹配失败的上限
	yearMax := t.Year() + 2

	// 设定用于年份判断的位
	yearBits := st.Year[0]
	delta := t.Year() - 1970
	if delta >= 64 {
		yearBits = st.Year[1]
		delta = delta - 64
	}

	// 防止以下情况的出现：因时间精度问题，10.001 秒的时候进入该方法，
	// 如果直接进行匹配，则第 10 秒的时间就会忽略
	t = t.Add(time.Second - time.Duration(t.Nanosecond())*time.Nanosecond)
	added := false

LOOP:
	// 超过匹配年限则返回零值时间
	if t.Year() > yearMax {
		return time.Time{}
	}

	for (1<<delta)&yearBits == 0 {
		if !added {
			added = true
			t = time.Date(t.Year(), time.January, 1, 0, 0, 0, 0, loc)
		}
		t = t.AddDate(1, 0, 0)

		if t.Year() > yearMax {
			return time.Time{}
		}
	}

	for (1<<t.Month())&st.Month == 0 {
		if !added {
			added = true
			t = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, loc)
		}
		t = t.AddDate(0, 1, 0)

		if t.Month() == time.January {
			goto LOOP
		}
	}

	// 不支持 DST（夏令时）
	for !isDayMatch(st, t) {
		if !added {
			added = true
			t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
		}
		t = t.AddDate(0, 0, 1)

		if t.Day() == 1 {
			goto LOOP
		}
	}

	for (1<<t.Hour())&st.Hour == 0 {
		if !added {
			added = true
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, loc)
		}
		t = t.Add(time.Hour)

		if t.Hour() == 0 {
			goto LOOP
		}
	}

	for (1<<t.Minute())&st.Minute == 0 {
		if !added {
			added = true
			t = t.Truncate(time.Minute)
		}
		t = t.Add(time.Minute)

		if t.Minute() == 0 {
			goto LOOP
		}
	}

	for (1<<t.Second())&st.Second == 0 {
		if !added {
			added = true
			t = t.Truncate(time.Second)
		}
		t = t.Add(time.Second)

		if t.Second() == 0 {
			goto LOOP
		}
	}

	return t.In(origLocation)
}

// 判断“日”是否匹配，匹配规则为：必须“日”和“星期”都匹配，则认为匹配
func isDayMatch(st *SchedTime, t time.Time) bool {
	domMatch := ((1 << t.Day()) & st.Dom) != 0
	dowMatch := ((1 << t.Weekday()) & st.Dow) != 0

	return domMatch && dowMatch
}
