package lib

import (
	"github.com/Netflix/go-env"
	"github.com/joho/godotenv"
)

type Environment struct {
	WebServer struct {
		Host     string `env:"WEB_HOST,default=0.0.0.0"`
		Port     int    `env:"WEB_PORT,default=8080"`
		TlsDir   string `env:"WEB_TLS_DIR"`
		Username string `env:"WEB_USERNAME,required=true"`
		Password string `env:"WEB_PASSWORD,required=true"`
	}

	// Proxmox API
	Proxmox struct {
		Host    string `env:"PROXMOX_HOST,required=true"`
		TokenID string `env:"PROXMOX_API_TOKEN_ID,required=true"`
		Secret  string `env:"PROXMOX_API_TOKEN_SECRET,required=true"`
	}

	// SSH Keys
	SSH struct {
		PublicKeyPath  string `env:"SSH_PUBLIC_KEY,required=true"`
		PrivateKeyPath string `env:"SSH_PRIVATE_KEY,required=true"`
	}

	// Container configs
	Container struct {
		HostnamePrefix string `env:"CONTAINER_HOSTNAME_PREFIX,required=true"`
		RootPassword   string `env:"CONTAINER_ROOT_PASSWORD,required=true"`
		StorageGB      int    `env:"CONTAINER_STORAGE_GB,required=true"`
		MemoryMB       int    `env:"CONTAINER_MEMORY_MB,required=true"`
		Cores          int    `env:"CONTAINER_CPU_CORES,required=true"`
		Template       string `env:"CONTAINER_TEMPLATE,required=true"`
		StoragePool    string `env:"CONTAINER_STORAGE_POOL,required=true"`
		GatewayIPv4    string `env:"CONTAINER_GATEWAY,required=true"`
		IndividualCIDR int    `env:"CONTAINER_CIDR,required=true"`
		Nameserver     string `env:"CONTAINER_NAMESERVER,required=true"`
		SearchDomain   string `env:"CONTAINER_SEARCH_DOMAIN,required=true"`
	}

	Database struct {
		File      string `env:"DB_FILE,default=opnlaas.db"`
		Salt      string `env:"DB_SALT,required=true"`
		QueueSize int    `env:"DB_QUEUE_SIZE,default=256"`
	}
}

var Config Environment
var LocalIP string

func InitEnv() error {
	if err := godotenv.Load(); err != nil {
		return err
	}

	_, err := env.UnmarshalFromEnviron(&Config)
	if err != nil {
		return err
	}

	LocalIP, err = GetLocalIP()

	if err != nil {
		return err
	}

	return nil
}
