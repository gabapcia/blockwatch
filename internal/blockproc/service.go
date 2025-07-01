package blockproc

type Service interface {
}

type service struct {
}

var _ Service = (*service)(nil)

func New() *service {
	return &service{}
}
