package ashara

type Client interface {
}

type BaseClient struct {
}

func NewBaseClient() Client {
	return &BaseClient{}
}
