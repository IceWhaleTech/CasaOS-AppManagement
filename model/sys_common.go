package model

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
	HttpPort     string
	RunMode      string
	ServerApi    string
	LockAccount  bool
	Token        string
	USBAutoMount string
	SocketPort   string
	UpdateUrl    string
}

type CasaOSGlobalVariables struct {
	AppChange bool
}
