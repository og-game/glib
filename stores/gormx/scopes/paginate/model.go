package paginate

// Pagination 分页结构体
type Pagination struct {
	Page          int   `json:"page"`
	PageSize      int   `json:"page_size"`
	Total         int64 `json:"total"`
	Rows          any   `json:"rows"`
	Extend        any   `json:"extend,omitempty"`
	TotalPage     int64 `json:"total_page"`
	NotQueryTotal bool  `json:"not_query_total"` // 是否不查询总数
	ForcePageSize bool  `json:"force_page_size"` // 是否强制获取超出限制的条数
}

// Default constants
const (
	DefaultPage     = 1
	DefaultPageSize = 10
	MaxPageSize     = 100
)

// Offset 计算分页偏移量
func (p *Pagination) Offset() int {
	page := p.Page
	if page <= 0 {
		page = DefaultPage
	}
	return (page - 1) * p.Limit()
}

// Limit 获取分页条数限制
func (p *Pagination) Limit() int {
	size := p.PageSize
	if size <= 0 {
		size = DefaultPageSize
	}
	if !p.ForcePageSize && size > MaxPageSize {
		size = MaxPageSize
	}
	return size
}

// GetPage 返回当前页码
func (p *Pagination) GetPage() int {
	if p.Page <= 0 {
		return DefaultPage
	}
	return p.Page
}

// GetPageSize 返回每页条数
func (p *Pagination) GetPageSize() int {
	return p.Limit()
}
