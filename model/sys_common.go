package model

type CommonModel struct {
	RuntimePath string
}

type APPModel struct {
	LogPath        string
	LogSaveName    string
	LogFileExt     string
	DateStrFormat  string
	DateTimeFormat string
	UserDataPath   string
	TimeFormat     string
	DateFormat     string
	DBPath         string
	ShellPath      string
}

type ServerModel struct {
	HTTPPort     string
	RunMode      string
	ServerAPI    string
	LockAccount  bool
	Token        string
	USBAutoMount string
	SocketPort   string
	UpdateURL    string
}

type CasaOSGlobalVariables struct {
	AppChange bool
}
