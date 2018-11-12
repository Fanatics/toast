package base

type Data struct {
	SchemaVersion string
	*AuditLog
}

func (d Data) Export() bool { return true }

func (d Data) Namespace() string { return "com.fanatics.amdm" }

func (d Data) Validate() error { return nil }

type AuditLog struct {
	CreatedAt int64 // unix nano
	UpdatedAt int64 // unix nano
	CreatedBy string
	UpdatedBy string
}

type EmbedMe interface {
	// Internal is a method from the embedded interface
	Internal(string) error
}
