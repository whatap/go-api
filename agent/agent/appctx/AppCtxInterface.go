package appctx

type IAppCtx interface {
	Name() string
	Parse(hash uint32, url string) string // ctx 파라미터 제거
	Update()
}

type NameAndUrl struct {
	Name string
	URL  string
}
