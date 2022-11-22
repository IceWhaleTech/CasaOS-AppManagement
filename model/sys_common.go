package model

type CommonModel struct {
	RuntimePath string
}

type APPModel struct {
	LogPath     string
	LogSaveName string
	LogFileExt  string
	DBPath      string
}

type ServerModel struct {
	ServerAPI string
}

type CasaOSGlobalVariables struct {
	AppChange bool
}
