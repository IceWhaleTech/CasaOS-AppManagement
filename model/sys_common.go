package model

type CommonModel struct {
	RuntimePath string
}

type APPModel struct {
	LogPath      string
	LogSaveName  string
	LogFileExt   string
	DBPath       string
	AppStorePath string
}

type ServerModel struct {
	ServerAPI    string
	AppStoreList []string `ini:"appstore,,allowshadow"`
}

type CasaOSGlobalVariables struct {
	AppChange bool
}
