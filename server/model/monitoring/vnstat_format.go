package monitoring

import "time"

// VnStatResponse vnStat JSON 响应的通用结构
type VnStatResponse struct {
	VnStatVersion string                `json:"vnstatversion"`
	JsonVersion   string                `json:"jsonversion"`
	Interfaces    []VnStatInterfaceData `json:"interfaces"`
}

// VnStatInterfaceData vnStat 接口数据
type VnStatInterfaceData struct {
	Name    string            `json:"name"`
	Alias   string            `json:"alias,omitempty"`
	Created *VnStatTimestamp  `json:"created,omitempty"` // v2+ 才有
	Updated *VnStatTimestamp  `json:"updated,omitempty"` // v2+ 才有
	Traffic VnStatTrafficData `json:"traffic"`
}

// VnStatTimestamp 时间戳信息
type VnStatTimestamp struct {
	Date      VnStatDate  `json:"date"`
	Time      *VnStatTime `json:"time,omitempty"`      // 有些地方没有时间
	Timestamp *int64      `json:"timestamp,omitempty"` // v2+ 才有
}

// VnStatDate 日期信息
type VnStatDate struct {
	Year  int `json:"year"`
	Month int `json:"month,omitempty"`
	Day   int `json:"day,omitempty"`
}

// VnStatTime 时间信息
type VnStatTime struct {
	Hour   int `json:"hour"`
	Minute int `json:"minute"`
}

// VnStatTrafficData 流量数据
type VnStatTrafficData struct {
	Total VnStatTotal `json:"total"`
	// v1 格式字段 (复数形式)
	Days   []VnStatDayRecord   `json:"days,omitempty"`   // v1 使用
	Months []VnStatMonthRecord `json:"months,omitempty"` // v1 使用
	Tops   []VnStatTopRecord   `json:"tops,omitempty"`   // v1 使用
	// v2 格式字段 (单数形式)
	FiveMinute []VnStatFiveMinRecord `json:"fiveminute,omitempty"` // v2+ 新增
	Hour       []VnStatHourRecord    `json:"hour,omitempty"`       // v2+ 新增
	Day        []VnStatDayRecord     `json:"day,omitempty"`        // v2 使用
	Month      []VnStatMonthRecord   `json:"month,omitempty"`      // v2 使用
	Year       []VnStatYearRecord    `json:"year,omitempty"`       // v2+ 新增
	Top        []VnStatTopRecord     `json:"top,omitempty"`        // v2 使用
}

// VnStatTotal 总流量
type VnStatTotal struct {
	Rx int64 `json:"rx"`
	Tx int64 `json:"tx"`
}

// VnStatFiveMinRecord 5分钟流量记录 (v2+)
type VnStatFiveMinRecord struct {
	ID        *int        `json:"id,omitempty"`
	Date      VnStatDate  `json:"date"`
	Time      *VnStatTime `json:"time,omitempty"`
	Timestamp *int64      `json:"timestamp,omitempty"`
	Rx        int64       `json:"rx"`
	Tx        int64       `json:"tx"`
}

// VnStatHourRecord 小时流量记录 (v2+)
type VnStatHourRecord struct {
	ID        *int        `json:"id,omitempty"`
	Date      VnStatDate  `json:"date"`
	Time      *VnStatTime `json:"time,omitempty"`
	Timestamp *int64      `json:"timestamp,omitempty"`
	Rx        int64       `json:"rx"`
	Tx        int64       `json:"tx"`
}

// VnStatDayRecord 日流量记录
type VnStatDayRecord struct {
	ID        *int       `json:"id,omitempty"` // v2+ 才有
	Date      VnStatDate `json:"date"`
	Timestamp *int64     `json:"timestamp,omitempty"` // v2+ 才有
	Rx        int64      `json:"rx"`
	Tx        int64      `json:"tx"`
}

// VnStatMonthRecord 月流量记录
type VnStatMonthRecord struct {
	ID        *int       `json:"id,omitempty"`        // v2+ 才有
	Date      VnStatDate `json:"date"`                // 只有年月
	Timestamp *int64     `json:"timestamp,omitempty"` // v2+ 才有
	Rx        int64      `json:"rx"`
	Tx        int64      `json:"tx"`
}

// VnStatYearRecord 年流量记录 (v2+)
type VnStatYearRecord struct {
	ID        *int       `json:"id,omitempty"`
	Date      VnStatDate `json:"date"` // 只有年
	Timestamp *int64     `json:"timestamp,omitempty"`
	Rx        int64      `json:"rx"`
	Tx        int64      `json:"tx"`
}

// VnStatTopRecord 峰值流量记录
type VnStatTopRecord struct {
	ID        *int       `json:"id,omitempty"` // v2+ 才有
	Date      VnStatDate `json:"date"`
	Timestamp *int64     `json:"timestamp,omitempty"` // v2+ 才有
	Rx        int64      `json:"rx"`
	Tx        int64      `json:"tx"`
}

// GetNormalizedTrafficData 获取标准化的流量数据（兼容v1和v2）
func (t *VnStatTrafficData) GetNormalizedTrafficData() NormalizedTrafficData {
	result := NormalizedTrafficData{
		Total: t.Total,
	}

	// 优先使用v2格式，如果没有则使用v1格式
	if len(t.Day) > 0 {
		result.Days = t.Day
	} else if len(t.Days) > 0 {
		result.Days = t.Days
	}

	if len(t.Month) > 0 {
		result.Months = t.Month
	} else if len(t.Months) > 0 {
		result.Months = t.Months
	}

	if len(t.Top) > 0 {
		result.Tops = t.Top
	} else if len(t.Tops) > 0 {
		result.Tops = t.Tops
	}

	// v2+ 特有的数据
	result.FiveMinute = t.FiveMinute
	result.Hour = t.Hour
	result.Year = t.Year

	return result
}

// NormalizedTrafficData 标准化后的流量数据
type NormalizedTrafficData struct {
	Total      VnStatTotal           `json:"total"`
	FiveMinute []VnStatFiveMinRecord `json:"fiveminute,omitempty"`
	Hour       []VnStatHourRecord    `json:"hour,omitempty"`
	Days       []VnStatDayRecord     `json:"days"`
	Months     []VnStatMonthRecord   `json:"months"`
	Year       []VnStatYearRecord    `json:"year,omitempty"`
	Tops       []VnStatTopRecord     `json:"tops"`
}

// GetTimestamp 获取时间戳，如果没有则根据日期计算
func (d *VnStatDayRecord) GetTimestamp() int64 {
	if d.Timestamp != nil {
		return *d.Timestamp
	}
	// 如果没有时间戳，根据日期计算
	t := time.Date(d.Date.Year, time.Month(d.Date.Month), d.Date.Day, 0, 0, 0, 0, time.UTC)
	return t.Unix()
}

// GetTimestamp 获取时间戳，如果没有则根据日期计算
func (m *VnStatMonthRecord) GetTimestamp() int64 {
	if m.Timestamp != nil {
		return *m.Timestamp
	}
	// 如果没有时间戳，根据日期计算（月初）
	t := time.Date(m.Date.Year, time.Month(m.Date.Month), 1, 0, 0, 0, 0, time.UTC)
	return t.Unix()
}

// GetTimestamp 获取时间戳，如果没有则根据日期计算
func (y *VnStatYearRecord) GetTimestamp() int64 {
	if y.Timestamp != nil {
		return *y.Timestamp
	}
	// 如果没有时间戳，根据日期计算（年初）
	t := time.Date(y.Date.Year, time.January, 1, 0, 0, 0, 0, time.UTC)
	return t.Unix()
}

// GetTimestamp 获取时间戳，如果没有则根据日期计算
func (top *VnStatTopRecord) GetTimestamp() int64 {
	if top.Timestamp != nil {
		return *top.Timestamp
	}
	// 如果没有时间戳，根据日期计算
	day := top.Date.Day
	if day == 0 {
		day = 1
	}
	month := top.Date.Month
	if month == 0 {
		month = 1
	}
	t := time.Date(top.Date.Year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	return t.Unix()
}
