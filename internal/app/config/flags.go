package config

import (
	"flag"
	"time"
)

// AppFlags содержит установленные пользователем флаги. Если флаг не был явно задан,
// поле является nil.
type AppFlags struct {
	ConfigFileName      *string
	BaseURL             *string
	SrvAddr             *string
	Secret              *string
	DBType              *string
	StorageFileName     *string
	DSN                 *string
	EnableHTTPS         *bool
	TrustedSubnet       *string
	DeleteFlushInterval *time.Duration
}

// GetFlags считывает установленные пользователем флаги и возвращает структуру AppFlags.
func GetFlags() AppFlags {
	var flushInterval int
	configFilename := flag.String("c", DefaultCfgFileName, "configuration file")
	srvAddr := flag.String("a", srvAddrDefault, "Server address")
	baseURL := flag.String("b", baseURLDefault, "Base URL")
	secret := flag.String("p", "*****", "Secret key for hashing cookies") // чтобы ключ по умолчанию не отображался в usage, придется действовать из-за угла))
	dbType := flag.String("r", DBInmem, "Storage type (default inmem)\n- inmem\t\tin-memory storage periodically written to .gob file\n"+
		"- postgres\tPostgreSQL database")
	storageFileName := flag.String("f", fileStorageDefault, "File storage path")
	dsn := flag.String("d", "", "Database DSN")
	enableHTTPS := flag.Bool("s", false, "enable HTTPS")
	trustedSubnet := flag.String("t", "", "Trusted subnet for internal requests")
	flag.IntVar(&flushInterval, "F", int(defaultDeleteFlushInterval/time.Millisecond), "Flush interval for accumulate data to delete in milliseconds")
	flag.Parse()

	delFlushInterval := time.Duration(flushInterval) * time.Millisecond

	sf := AppFlags{}
	flag.Visit(func(f *flag.Flag) { // установить только те поля структуры setFlags, которые были заданы явно
		switch f.Name {
		case "c":
			sf.ConfigFileName = configFilename
		case "a":
			sf.SrvAddr = srvAddr
		case "b":
			sf.BaseURL = baseURL
		case "p":
			sf.Secret = secret
		case "r":
			sf.DBType = dbType
		case "f":
			sf.StorageFileName = storageFileName
		case "d":
			sf.DSN = dsn
		case "s":
			sf.EnableHTTPS = enableHTTPS
		case "F":
			sf.DeleteFlushInterval = &delFlushInterval
		case "t":
			sf.TrustedSubnet = trustedSubnet
		}
	})

	return sf
}
