package elect

// Option 选举配置选项
type Option func(*Options)

// Options 选举选项
type Options struct {
	OnLeader func() // 成为 Leader 时回调
	OnDemote func() // 失去 Leader 时回调
}

// WithOnLeader 设置成为 Leader 时的回调
func WithOnLeader(fn func()) Option {
	return func(o *Options) {
		o.OnLeader = fn
	}
}

// WithOnDemote 设置失去 Leader 时的回调
func WithOnDemote(fn func()) Option {
	return func(o *Options) {
		o.OnDemote = fn
	}
}
