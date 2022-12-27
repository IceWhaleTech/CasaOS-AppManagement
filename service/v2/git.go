package v2

type Git struct {
	appStoreList map[string]*AppStore
}

func NewGitService() *Git {
	return &Git{
		appStoreList: make(map[string]*AppStore),
	}
}
