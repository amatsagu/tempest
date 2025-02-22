package ashara

// compile-time interface assertion
var _ Client = (*BaseClient)(nil)

type Client interface {
}

type BaseClient struct {
}

func NewBaseClient() Client {
	return &BaseClient{}
}
